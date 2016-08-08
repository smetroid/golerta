package ldap

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/kataras/iris"
	"gopkg.in/ldap.v2"
	"time"
)

type LDAPAuthProvider struct {
	conn         *ldap.Conn
	signingKey   string
	Host         string   `toml:"host"`
	Port         int      `toml:"port"`
	UseSSL       bool     `toml:"use_ssl"`
	BaseDN       string   `toml:"base_dn"`
	BindDN       string   `toml:"bind_dn"`
	BindPassword string   `toml:"bind_password"`
	UserFilter   string   `toml:"user_filter"`
	Attributes   []string `toml:"attributes"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

//JSON struct that holds generated authorization token
type AuthToken struct {
	Token string `json:"token"`
}

//JSON struct for login errors
type LoginError struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Handles login request
func (lc *LDAPAuthProvider) LoginHandler(ctx *iris.Context) {
	loginRequest := LoginRequest{}
	err := ctx.ReadJSON(&loginRequest)

	if err != nil || loginRequest.Username == "" || loginRequest.Password == "" {
		ctx.JSON(iris.StatusUnauthorized, LoginError{"error", "Invalid login request"})
		return
	}

	loginSuccess, err := lc.Authenticate(loginRequest.Username, loginRequest.Password)

	if err != nil || !loginSuccess {
		ctx.JSON(iris.StatusUnauthorized, LoginError{"error", "Login failed"})
		return
	}

	token, _ := lc.createToken(loginRequest.Username)
	authToken := AuthToken{token}
	ctx.JSON(iris.StatusOK, authToken)
}

func (lc *LDAPAuthProvider) createToken(username string) (string, error) {
	mySigningKey := []byte(lc.signingKey)
	expirationTimestamp := time.Now().Add(time.Hour * 48).Unix()
	claims := jwt.MapClaims{
		"jti": username,
		"iss": "ldap",
		"exp": expirationTimestamp,
		"name": username,
		//Everyone who logs in is an admin by default for now. Could check ldap groups for this.
		"role": "admin",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	ss, err := token.SignedString(mySigningKey)
	if err != nil {
		return "", err
	}
	return ss, nil
}

func (lc *LDAPAuthProvider) SetSigningKey(key string) {
	lc.signingKey = key
}

// Connect connects to the ldap backend
func (lc *LDAPAuthProvider) Connect() error {
	if lc.conn == nil {
		var l *ldap.Conn
		var err error
		address := fmt.Sprintf("%s:%d", lc.Host, lc.Port)
		if !lc.UseSSL {
			l, err = ldap.Dial("tcp", address)
			if err != nil {
				return err
			}

			// Reconnect with TLS
			err = l.StartTLS(&tls.Config{InsecureSkipVerify: true})
			if err != nil {
				return err
			}
		} else {
			l, err = ldap.DialTLS("tcp", address, &tls.Config{InsecureSkipVerify: false})
			if err != nil {
				return err
			}
		}

		lc.conn = l
	}
	return nil
}

// Close closes the ldap backend connection
func (lc *LDAPAuthProvider) Close() {
	if lc.conn != nil {
		lc.conn.Close()
		lc.conn = nil
	}
}

// Authenticate authenticates the user against the ldap backend
func (lc *LDAPAuthProvider) Authenticate(username, password string) (bool, error) {
	err := lc.Connect()
	if err != nil {
		return false, err
	}

	// First bind with a read only user
	if lc.BindDN != "" && lc.BindPassword != "" {
		err := lc.conn.Bind(lc.BindDN, lc.BindPassword)
		if err != nil {
			return false, err
		}
	}

	attributes := append(lc.Attributes, "dn")
	// Search for the given username
	searchRequest := ldap.NewSearchRequest(
		lc.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf(lc.UserFilter, username),
		attributes,
		nil,
	)

	sr, err := lc.conn.Search(searchRequest)
	if err != nil {
		return false, err
	}

	if len(sr.Entries) < 1 {
		return false, errors.New("User does not exist")
	}

	if len(sr.Entries) > 1 {
		return false, errors.New("Too many entries returned")
	}

	userDN := sr.Entries[0].DN
	user := map[string]string{}
	for _, attr := range lc.Attributes {
		user[attr] = sr.Entries[0].GetAttributeValue(attr)
	}

	// Bind as the user to verify their password
	err = lc.conn.Bind(userDN, password)
	if err != nil {
		return false, err
	}

	// Rebind as the read only user for any further queries
	if lc.BindDN != "" && lc.BindPassword != "" {
		err = lc.conn.Bind(lc.BindDN, lc.BindPassword)
		if err != nil {
			return true, err
		}
	}

	return true, nil
}

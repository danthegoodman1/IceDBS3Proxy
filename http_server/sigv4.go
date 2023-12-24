package http_server

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	"sort"
	"strings"
)

var (
	ErrInvalidSignature = echo.NewHTTPError(403, "invalid signature")
)

func getHMAC(key []byte, data []byte) []byte {
	hash := hmac.New(sha256.New, key)
	hash.Write(data)
	return hash.Sum(nil)
}

func getSHA256(data []byte) []byte {
	hash := sha256.New()
	hash.Write(data)
	return hash.Sum(nil)
}

func getCanonicalRequest(c echo.Context) string {
	s := ""
	s += c.Request().Method + "\n"
	s += c.Request().URL.EscapedPath() + "\n"
	s += c.Request().URL.Query().Encode() + "\n"

	signedHeadersList, _ := lo.Find(strings.Split(c.Request().Header.Get("Authorization"), ", "), func(item string) bool {
		return strings.HasPrefix(item, "SignedHeaders")
	})

	signedHeaders := strings.Split(strings.ReplaceAll(strings.ReplaceAll(signedHeadersList, "SignedHeaders=", ""), ",", ""), ";")
	sort.Strings(signedHeaders) // must be sorted alphabetically
	for _, header := range signedHeaders {
		if header == "host" {
			// For some reason the host header was blank (thanks echo?)
			s += strings.ToLower(header) + ":" + strings.TrimSpace(c.Request().Host) + "\n"
			continue
		}
		s += strings.ToLower(header) + ":" + strings.TrimSpace(c.Request().Header.Get(header)) + "\n"
	}

	s += "\n" // examples have this JESUS WHY DOCS FFS

	s += strings.Join(signedHeaders, ";") + "\n"

	shaHeader := c.Request().Header.Get("x-amz-content-sha256")
	s += lo.Ternary(shaHeader == "", "UNSIGNED-PAYLOAD", shaHeader)

	return s
}

func getStringToSign(c echo.Context, canonicalRequest string) string {
	s := "AWS4-HMAC-SHA256" + "\n"
	s += c.Request().Header.Get("X-Amz-Date") + "\n"

	scope := c.Request().Header.Get("X-Amz-Date")[:8] + "/" + "us-east-1" + "/" + "dynamodb" + "/aws4_request"
	s += scope + "\n"
	s += fmt.Sprintf("%x", getSHA256([]byte(canonicalRequest)))

	return s
}

func getSigningKey(c echo.Context, password string) []byte {
	dateKey := getHMAC([]byte("AWS4"+password), []byte(c.Request().Header.Get("X-Amz-Date")[:8]))
	dateRegionKey := getHMAC(dateKey, []byte("us-east-1"))
	dateRegionServiceKey := getHMAC(dateRegionKey, []byte("dynamodb"))
	signingKey := getHMAC(dateRegionServiceKey, []byte("aws4_request"))
	return signingKey
}

type (
	AWSAuthHeader struct {
		Credential    AWSAuthHeaderCredential
		SignedHeaders []string
		Signature     string
	}

	AWSAuthHeaderCredential struct {
		KeyID   string
		Date    string
		Region  string
		Service string
		Request string
	}
)

func parseAuthHeader(header string) AWSAuthHeader {
	var authHeader AWSAuthHeader
	parts := strings.Split(header, " ")
	for _, part := range parts {
		// Remove the trailing `,`
		if part[len(part)-1] == ',' {
			part = part[:len(part)-1]
		}
		fmt.Println("Got part", part)
		keyValue := strings.SplitN(part, "=", 2)
		if len(keyValue) != 2 {
			continue
		}

		key, value := keyValue[0], keyValue[1]
		switch key {
		case "Credential":
			credentialParts := strings.Split(value, "/")
			authHeader.Credential = AWSAuthHeaderCredential{
				KeyID:   credentialParts[0],
				Date:    credentialParts[1],
				Region:  credentialParts[2],
				Service: credentialParts[3],
				Request: credentialParts[4],
			}
		case "SignedHeaders":
			authHeader.SignedHeaders = strings.Split(value, ";")
		case "Signature":
			authHeader.Signature = value
		default:
			continue
		}
	}
	return authHeader
}

func verifyAWSRequest(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		parsedHeader := parseAuthHeader(c.Request().Header.Get("Authorization"))
		canonicalRequest := getCanonicalRequest(c)
		stringToSign := getStringToSign(c, canonicalRequest)
		signingKey := getSigningKey(c, "testpassword") // TODO: real password from key id
		signature := fmt.Sprintf("%x", getHMAC(signingKey, []byte(stringToSign)))

		if signature != parsedHeader.Signature {
			return ErrInvalidSignature
		}

		cc, _ := c.(*CustomContext)
		cc.AWSCredentials = parsedHeader.Credential

		return next(c)
	}
}

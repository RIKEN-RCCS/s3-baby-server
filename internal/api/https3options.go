// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	//"github.com/gorilla/mux"
)

type HTTPS3Options struct {
	Logger  *slog.Logger
	request *http.Request
}

func newHTTPS3Options(r *http.Request, logger *slog.Logger) *HTTPS3Options {
	return &HTTPS3Options{Logger: logger, request: r}
}

func (o *HTTPS3Options) GetOption(key string) string {
	if req := o.request.Header.Get(key); req != "" {
		return req
	}
	return o.request.URL.Query().Get(key)
}

func (o *HTTPS3Options) HeaderQueryCheck(param []string) bool {
	for _, h := range param {
		if o.GetOption(h) != "" {
			o.Logger.Debug("Invalid request：", "error", h)
			return false
		}
	}
	o.Logger.Debug("No Invalid request")
	return true
}

func (o *HTTPS3Options) GetBucket() string {
	//vars := mux.Vars(o.request)
	//return vars["bucket"]
	return o.request.PathValue("bucket")
}

func (o *HTTPS3Options) GetKey() string {
	//vars := mux.Vars(o.request)
	//return vars["key"]
	return o.request.PathValue("key")
}

func (o *HTTPS3Options) GetPath() string {
	path := filepath.Join(o.GetBucket(), o.GetKey())
	if isDir := strings.HasSuffix(o.GetKey(), "/"); isDir {
		path += string(filepath.Separator)
	}
	return path
}

func (o *HTTPS3Options) GetBody() []byte {
	body, _ := io.ReadAll(o.request.Body)
	return body
}

func (o *HTTPS3Options) Validate(params map[string]string) bool {
	for key, value := range params {
		v := o.GetOption(key)
		if v != value && v != "" { // 指定された値が許容する値じゃない場合にエラー
			return false
		}
	}
	return true
}

func (o *HTTPS3Options) CheckKeyPath(rootPath, path string) bool {
	rAPath, _ := filepath.Abs(rootPath)
	tAPath, err := filepath.Abs(filepath.Join(rootPath, path))
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rAPath, tAPath)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..") // サーブしたディレクトリ直下にあるか確認
}

func (o *HTTPS3Options) CheckErrorHeader() bool {
	errorHeader := []string{
		"fetch-owner",
		"versionId",
		"x-amz-acl",
		"x-amz-bucket-object-lock-enabled",
		"x-amz-bypass-governance-retention",
		"x-amz-copy-source-server-side-encryption-customer-algorithm",
		"x-amz-copy-source-server-side-encryption-customer-key",
		"x-amz-copy-source-server-side-encryption-customer-key-MD5",
		"x-amz-expected-bucket-owner",
		"x-amz-grant-full-control",
		"x-amz-grant-read",
		"x-amz-grant-read-acp",
		"x-amz-grant-write",
		"x-amz-grant-write-acp",
		"x-amz-if-match-initiated-time",
		"x-amz-if-match-last-modified-time",
		"x-amz-if-match-size",
		"x-amz-mfa",
		"x-amz-object-lock-legal-hold",
		"x-amz-object-lock-mode",
		"x-amz-object-lock-retain-until-date",
		"x-amz-server-side-encryption",
		"x-amz-server-side-encryption-aws-kms-key-id",
		"x-amz-server-side-encryption-bucket-key-enabled",
		"x-amz-server-side-encryption-context",
		"x-amz-server-side-encryption-customer-algorithm",
		"x-amz-server-side-encryption-customer-key",
		"x-amz-server-side-encryption-customer-key-MD5",
		"x-amz-source-expected-bucket-owner",
		"x-amz-write-offset-bytes",
	}
	return o.HeaderQueryCheck(errorHeader)
}

func createCanonicalRequest(req *http.Request, signedHeaders []string, hashedPayload string) string {
	canonicalURI := normalizeURIPath(req.URL.Path) // URLの正規化
	if canonicalURI == "" {
		canonicalURI = "/"
	}
	canonicalQueryString := ""
	if req.URL.RawQuery != "" {
		queryParts := strings.Split(req.URL.RawQuery, "&")
		var keyValuePairs []struct {
			key   string
			value string
		}
		for _, part := range queryParts {
			if part == "" {
				continue
			}
			var key, value string
			if eqIndex := strings.Index(part, "="); eqIndex != -1 {
				key = part[:eqIndex]
				value = part[eqIndex+1:]
			} else {
				key = part
				value = ""
			}
			decodedKey, err := url.QueryUnescape(key)
			if err != nil {
				decodedKey = key // デコードに失敗した場合は元のキーを保持
			}
			keyValuePairs = append(keyValuePairs, struct {
				key   string
				value string
			}{
				key:   decodedKey,
				value: value, // 値はエンコードされたまま保持
			})
		}
		sort.Slice(keyValuePairs, func(i, j int) bool {
			return keyValuePairs[i].key < keyValuePairs[j].key
		})
		var canonicalQueryParts []string
		for _, pair := range keyValuePairs {
			encodedKey := url.QueryEscape(pair.key)
			canonicalQueryParts = append(canonicalQueryParts, fmt.Sprintf("%s=%s", encodedKey, pair.value))
		}
		canonicalQueryString = strings.Join(canonicalQueryParts, "&")
	}
	sort.Strings(signedHeaders)
	var canonicalHeaderParts []string
	for _, headerName := range signedHeaders {
		headerValue := req.Header.Get(headerName)
		headerValue = strings.TrimSpace(headerValue)
		for strings.Contains(headerValue, "  ") {
			headerValue = strings.ReplaceAll(headerValue, "  ", " ")
		}
		canonicalHeaderParts = append(canonicalHeaderParts, fmt.Sprintf("%s:%s", headerName, headerValue))
	}
	canonicalHeaders := strings.Join(canonicalHeaderParts, "\n") + "\n"
	signedHeadersString := strings.Join(signedHeaders, ";")
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		req.Method,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeadersString,
		hashedPayload)
}

func normalizeURIPath(path string) string {
	if path == "" {
		return "/" // パスがない場合は"/"として処理
	}
	normalized := filepath.Clean(path)                     // ".",".."などを削除
	normalized = strings.ReplaceAll(normalized, "\\", "/") // Windows, Linuxどちらにも対応
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}
	segments := strings.Split(normalized, "/")
	var encodedSegments []string
	for _, segment := range segments {
		if segment == "" {
			encodedSegments = append(encodedSegments, "")
			continue
		}
		encoded := awsPathEscape(segment) // RFC 3986に従ったエンコーディング
		encodedSegments = append(encodedSegments, encoded)
	}
	result := strings.Join(encodedSegments, "/")
	if strings.HasSuffix(path, "/") && !strings.HasSuffix(result, "/") && result != "/" {
		result += "/"
	}
	return result
}

func awsPathEscape(s string) string {
	var buf bytes.Buffer
	for _, r := range s {
		switch {
		case (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'): // 英数字はそのまま
			buf.WriteRune(r)
		case r == '-' || r == '_' || r == '.' || r == '~': // RFC 3986に予約されていない文字もそのまま
			buf.WriteRune(r)
		default:
			if r < 0x80 {
				buf.WriteString(fmt.Sprintf("%%%02X", r)) // ASCII
			} else {
				utf8Bytes := []byte(string(r))
				for _, b := range utf8Bytes {
					buf.WriteString(fmt.Sprintf("%%%02X", b)) // バイト単位でエンコード
				}
			}
		}
	}
	return buf.String()
}

func (o *HTTPS3Options) checkAuthorization(req1 *http.Request, authKey string) bool {
	header1 := req1.Header.Get("Authorization")
	if header1 == "" {
		return false
	}
	var authPassed authorizationS3v4 = scanAwsAuthorization(header1)
	if authPassed.signature == "" {
		return false
	}
	for i, h := range authPassed.signedHeaders {
		authPassed.signedHeaders[i] = strings.ToLower(h)
	}
	service := authPassed.credential[3]
	region := authPassed.credential[2]
	dateString := fixXAmzDate(req1.Header.Get("X-Amz-Date"))
	date, err := time.Parse(time.RFC3339, dateString)
	if err != nil {
		return false
	}
	keyPair := strings.Split(authKey, ",")
	if len(keyPair) < 2 { // accessKey, secretAccessKeyのペアじゃなければfalse
		return false
	}
	accessKeyID := strings.TrimSpace(keyPair[0])
	secretAccessKey := strings.TrimSpace(keyPair[1])
	credentials := aws.Credentials{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
	}
	if credentials.AccessKeyID == "" || credentials.SecretAccessKey == "" {
		return false
	}
	hash := req1.Header.Get("X-Amz-Content-Sha256")
	if hash == "" {
		hash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	}
	req2 := *req1
	req2.Header = make(http.Header)
	for k, v := range req1.Header {
		if slices.Contains(authPassed.signedHeaders, strings.ToLower(k)) {
			req2.Header[k] = make([]string, len(v))
			copy(req2.Header[k], v)
		}
	}
	urlCopy := *req1.URL
	req2.URL = &urlCopy
	req2.URL.Path = req1.URL.Path
	req2.URL.RawPath = ""
	if fwdHost := req1.Header.Get("X-Forwarded-Host"); fwdHost != "" {
		req2.Host = fwdHost
		req2.Header.Set("Host", fwdHost)
	} else {
		req2.Host = req1.Host
		req2.Header.Set("Host", req1.Host)
	}
	if !slices.Contains(authPassed.signedHeaders, "content-length") {
		req2.ContentLength = -1
	}
	calculatedSignature := calcAWSSigV4Signature(&req2, credentials, service, region, date, hash, authPassed.signedHeaders)
	if calculatedSignature != authPassed.signature {
		o.Logger.Debug("Signature mismatch:",
			"Expected:", authPassed.signature,
			"Calculated:", calculatedSignature,
			"Original Path:", req1.URL.Path,
			"Canonical URI:", normalizeURIPath(req1.URL.Path))
	}
	return calculatedSignature == authPassed.signature
}

func calcAWSSigV4Signature(req *http.Request, credentials aws.Credentials, service, region string, signingTime time.Time, hashedPayload string, signedHeaders []string) string {
	canonicalRequest := createCanonicalRequest(req, signedHeaders, hashedPayload)
	algorithm := "AWS4-HMAC-SHA256"
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request",
		signingTime.UTC().Format("20060102"), region, service)
	sha256CanonicalRequest := fmt.Sprintf("%x", sha256.Sum256([]byte(canonicalRequest)))
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s", algorithm, signingTime.UTC().Format("20060102T150405Z"), credentialScope, sha256CanonicalRequest)
	dateKey := hmacSHA256([]byte("AWS4"+credentials.SecretAccessKey), signingTime.UTC().Format("20060102"))
	dateRegionKey := hmacSHA256(dateKey, region)
	dateRegionServiceKey := hmacSHA256(dateRegionKey, service)
	signingKey := hmacSHA256(dateRegionServiceKey, "aws4_request")
	signature := fmt.Sprintf("%x", hmacSHA256(signingKey, stringToSign))
	return signature
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func fixXAmzDate(dateStr string) string {
	// RFC3339に変換
	if len(dateStr) == 16 && strings.HasSuffix(dateStr, "Z") {
		if len(dateStr) >= 15 {
			year := dateStr[0:4]
			month := dateStr[4:6]
			day := dateStr[6:8]
			hour := dateStr[9:11]
			minute := dateStr[11:13]
			second := dateStr[13:15]
			return fmt.Sprintf("%s-%s-%sT%s:%s:%sZ", year, month, day, hour, minute, second)
		}
	}
	return dateStr
}

type authorizationS3v4 struct {
	credential    [5]string
	signedHeaders []string
	signature     string
}

const (
	s3v4AuthorizationMethod = "AWS4-HMAC-SHA256"
)

func checkAllDigits(s string) bool {
	var re = regexp.MustCompile(`^[0-9]+$`)
	return re.MatchString(s)
}

func scanAwsAuthorization(auth string) authorizationS3v4 {
	var requiredHeaders = [3]string{"Host", "X-Amz-Content-Sha256", "X-Amz-Date"}
	var bad = authorizationS3v4{}
	if auth == "" {
		return bad
	}
	var i1 = strings.Index(auth, " ")
	if i1 == -1 || i1 != 16 {
		return bad
	}
	if auth[:16] != s3v4AuthorizationMethod {
		return bad
	}
	var slots [][2]string
	for _, s1 := range strings.Split(auth[16:], ",") {
		var s2 = strings.TrimSpace(s1)
		var i2 = strings.Index(s2, "=")
		if i2 == -1 || i2 == 0 || i2 == (len(s2)-1) {
			continue
		}
		slots = append(slots, [2]string{s2[:i2], s2[i2+1:]})
	}
	if len(slots) != 3 {
		return bad
	}
	var v = authorizationS3v4{}
	for _, kv := range slots {
		switch kv[0] {
		case "Credential":
			// "Credential={key}/20240511/us-east-1/s3/aws4_request"
			var c1 = strings.Split(kv[1], "/")
			if len(c1) != 5 {
				return bad
			}
			var c2 = [5]string(c1)
			if len(c2[1]) != 8 || !checkAllDigits(c2[1]) {
				return bad
			}
			if c2[3] != "s3" {
				return bad
			}
			if c2[4] != "aws4_request" {
				return bad
			}
			v.credential = c2
		case "SignedHeaders":
			// SignedHeaders=host;x-amz-content-sha256;x-amz-date
			var headers []string
			for _, h1 := range strings.Split(kv[1], ";") {
				headers = append(headers, http.CanonicalHeaderKey(h1))
			}
			for _, h2 := range requiredHeaders {
				if slices.Index(headers, h2) == -1 {
					return bad
				}
			}
			v.signedHeaders = headers
		case "Signature":
			v.signature = kv[1]
		default:
			return bad
		}
	}
	if v.credential == [5]string{} ||
		v.signedHeaders == nil ||
		v.signature == "" {
		return bad
	}
	return v
}

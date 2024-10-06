package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/snowflake"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v4"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/rs/cors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var node *snowflake.Node
var (
	ctx = context.Background()
	rdb *redis.Client
)

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

var jwtKey = []byte("80BEB12D58BC822705B6000584249652")

func init() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
}

func qrcodeHandler(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get("uuid")

	parts, err := getParsedDataFromRedis(uuid)
	if err != nil {
		fmt.Println(err)
		uuid = node.Generate().String()
	} else {
		expireTimestamp := parts[2]
		if isTimestampExpired(expireTimestamp) {
			err = rdb.Del(ctx, uuid).Err()
			if err != nil {
				return
			}
			uuid = node.Generate().String()
		}
	}

	qrcodeURL := "http://localhost:8080/MobileVerify?uuid=" + uuid

	png, err := qrcode.Encode(qrcodeURL, qrcode.Medium, 256)
	if err != nil {
		fmt.Println("无法生成二维码", err)
		return
	}

	base64Str := base64.StdEncoding.EncodeToString(png)

	data := struct {
		UUID   string `json:"uuid"`
		QRCode string `json:"qrcode"`
	}{
		UUID:   uuid,
		QRCode: base64Str,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	content := string(jsonData)

	storeDataInRedis(uuid)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(content))
}

func qrcodeLoginHandler(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get("uuid")
	userid := r.URL.Query().Get("userid")

	var content string

	parts, err := getParsedDataFromRedis(uuid)
	if err != nil {
		fmt.Println(err)
		content = `{"status": "fail","msg": "uuid has expired"}`
	} else {
		expireTimestamp := parts[2]
		if isTimestampExpired(expireTimestamp) {
			err = rdb.Del(ctx, uuid).Err()
			if err != nil {
				return
			}
			content = `{"status": "fail","msg": "uuid has expired"}`
		} else {
			redisTimestamp := parts[2]

			dataToRedis := fmt.Sprintf("%s-%s-%s", uuid, userid, redisTimestamp)

			redisErr := rdb.Set(ctx, uuid, dataToRedis, 0).Err()
			if redisErr != nil {
				fmt.Println("Error setting value in Redis:", redisErr)
				return
			}

			content = `{"status": "ok"}`
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(content))
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get("uuid")

	parts, err := getParsedDataFromRedis(uuid)
	if err != nil {
		fmt.Println(err)
		content := `{"status": "fail", "msg": "uuid has expired"}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(content))
		return
	}

	var content string
	if parts[1] != "0" {
		token := login(parts[1])
		content = `{"status": "ok", "token": "` + token + `"}`
	} else {
		content = `{"status": "fail", "msg": "userid not received"}`
	}

	expireTimestamp := parts[2]

	if isTimestampExpired(expireTimestamp) {
		content = `{"status": "fail", "msg": "uuid has expired"}`
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(content))
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	var content string
	if validateToken(token) {
		content = `{"status": "ok"}`
	} else {
		content = `{"status": "fail"}`
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(content))
}

func getUuidHandler(w http.ResponseWriter, r *http.Request) {
	uuid := node.Generate().String()

	content := `{"uuid": "` + uuid + `"}`

	storeDataInRedis(uuid)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(content))
}

func validateUuidHandler(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get("uuid")

	parts, err := getParsedDataFromRedis(uuid)
	if err != nil {
		fmt.Println(err)
		return
	}

	expireTimestamp := parts[2]

	var content string
	if isTimestampExpired(expireTimestamp) {
		content = `{"status": "fail"}`
		err = rdb.Del(ctx, uuid).Err()
		if err != nil {
			return
		}
	} else {
		content = `{"status": "ok"}`
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(content))
}

func isTimestampExpired(expireTimestamp string) bool {
	expireTime, err := strconv.ParseInt(expireTimestamp, 10, 64)
	if err != nil {
		return false
	}

	currentTime := time.Now().Unix()

	if currentTime > expireTime {
		return true
	} else {
		return false
	}
}

func main() {
	var err error
	node, err = snowflake.NewNode(1)
	if err != nil {
		fmt.Println("Error creating node:", err)
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/do/generateQRCode", qrcodeHandler)
	mux.HandleFunc("/do/qrcodeLogin", qrcodeLoginHandler)
	mux.HandleFunc("/do/getStatus", statusHandler)
	mux.HandleFunc("/do/validateToken", validateHandler)
	mux.HandleFunc("/do/getUuid", getUuidHandler)
	mux.HandleFunc("/do/validateUuid", validateUuidHandler)

	handler := cors.Default().Handler(mux)

	fmt.Println("Starting server at port 8081")
	httpErr := http.ListenAndServe(":8081", handler)
	if httpErr != nil {
		fmt.Println("Error starting server:", httpErr)
	}
}

func login(userid string) string {
	expirationTime := time.Now().Add(5 * time.Minute)
	claims := &Claims{
		Username: userid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		fmt.Println("生成 token 时出错:", err)
		return ""
	}

	return tokenString
}

func validateToken(tokenString string) bool {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			return false
		}
		return false
	}

	if !token.Valid {
		fmt.Println("无效的 token")
		return false
	}

	return true
}

func getParsedDataFromRedis(uuid string) ([]string, error) {
	val, err := rdb.Get(ctx, uuid).Result()
	if err != nil {
		return nil, fmt.Errorf("error getting value from Redis: %w. It is possible that the UUID has expired or is invalid", err)
	}

	parts := strings.Split(val, "-")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid data format in Redis")
	}

	return parts, nil
}

func storeDataInRedis(uuid string) error {
	timestamp := time.Now().Unix() + 30

	dataToRedis := fmt.Sprintf("%s-%s-%d", uuid, "0", timestamp)

	redisErr := rdb.Set(ctx, uuid, dataToRedis, 0).Err()
	if redisErr != nil {
		return fmt.Errorf("error setting value in Redis: %w", redisErr)
	}

	return nil
}

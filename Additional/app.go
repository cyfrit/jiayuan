package main

import (
	"fmt"
	"net/http"
	"context"
	"os"
	"strings"
	"encoding/base64"
	"encoding/json"
	"time"
	qrcode  "github.com/skip2/go-qrcode"
	"github.com/bwmarrin/snowflake"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v4"
)


var serverHost string = "http://10.0.0.1:8080" 

// 全局变量，用于存储 Snowflake 节点实例
var node *snowflake.Node
var (
	ctx = context.Background()
	rdb *redis.Client
)


// Claims 结构体
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// 定义用于签名 JWT 的密钥
var jwtKey = []byte("80BEB12D58BC822705B6000584249652")


func init() {
	// 初始化 Redis 客户端
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis 服务器地址
		Password: "",               // Redis 密码
		DB:       0,                // 数据库编号
	})
}

func readHTMLFile(filepath string) (string, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// loginHandler 处理 /login 路由
func loginHandler(w http.ResponseWriter, r *http.Request) {
	content, err := readHTMLFile("login.html")
	if err != nil {
		http.Error(w, "Could not read login.html", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(content))
}

func mobileHandler(w http.ResponseWriter, r *http.Request) {
	content, err := readHTMLFile("MobileVerify.html")
	if err != nil {
		http.Error(w, "Could not read MobileVerify.html", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(content))
}

func qrcodeHandler(w http.ResponseWriter, r *http.Request) {
	uuid := node.Generate().String()

	qrcodeURL := serverHost + "/MobileVerify?uuid=" + uuid

	// 生成二维码，返回一个字节数组
	png, err := qrcode.Encode(qrcodeURL, qrcode.Medium, 256)
	if err != nil {
		fmt.Println("无法生成二维码", err)
		return
	}

	// 将二维码字节数组转换为Base64字符串
	base64Str := base64.StdEncoding.EncodeToString(png)

	// 使用结构体来保证字段顺序
	data := struct {
		UUID   string `json:"uuid"`
		QRCode string `json:"qrcode"`
	}{
		UUID:   uuid,
		QRCode: base64Str,
	}

	// 将结构体转换为 JSON 字符串
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	content := string(jsonData)

	// 当前时间戳
	timestamp := time.Now().Unix()

	// 拼接数据
	dataToRedis := fmt.Sprintf("%s-%s-%d", uuid, "0", timestamp)

	// 将数据写入 Redis
	redisErr := rdb.Set(ctx, uuid, dataToRedis, 0).Err()
	if redisErr != nil {
		fmt.Println("Error setting value in Redis:", redisErr)
		return
	}

	//为了方便，没有完善的错误处理

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(content))
}


func qrcodeLoginHandler(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get("uuid")
	userid := r.URL.Query().Get("userid")

	// 从 Redis 中读取数据
	val, err := rdb.Get(ctx, uuid).Result()
	if err != nil {
		fmt.Println("Error getting value from Redis:", err)
		return
	}

	// 解析 Redis 中的数据
	parts := strings.Split(val, "-")
	if len(parts) != 3 {
		fmt.Println("Invalid data format in Redis")
		return
	}

	redisTimestamp := parts[2]

	dataToRedis := fmt.Sprintf("%s-%s-%s", uuid, userid, redisTimestamp)

	// 将数据写入 Redis
	redisErr := rdb.Set(ctx, uuid, dataToRedis, 0).Err()
	if redisErr != nil {
		fmt.Println("Error setting value in Redis:", redisErr)
		return
	}

	//为了方便，没有完善的错误处理


	content := `{"status": "ok"}`

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(content))
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get("uuid")

	// 从 Redis 中读取数据
	val, err := rdb.Get(ctx, uuid).Result()
	if err != nil {
		fmt.Println("Error getting value from Redis:", err)
		return
	}

	// 解析 Redis 中的数据
	parts := strings.Split(val, "-")
	if len(parts) != 3 {
		fmt.Println("Invalid data format in Redis")
		return
	}


	var content string
	if parts[1] != "0" {
		token := login(parts[1])
		content = `{"status": "ok", "token": "` + token + `"}`
	} else {
		content = `{"status": "fail"}`
	}

	// 为了方便，没有完善的错误处理


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

	// 为了方便，没有完善的错误处理


	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(content))
}

func main() {


	var err error
	// 创建 Snowflake 节点实例
	node, err = snowflake.NewNode(1)
	if err != nil {
		fmt.Println("Error creating node:", err)
		return
	}

	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/MobileVerify", mobileHandler)

	http.HandleFunc("/do/generateQRCode", qrcodeHandler)
	http.HandleFunc("/do/qrcodeLogin", qrcodeLoginHandler)
	http.HandleFunc("/do/getStatus", statusHandler)
	http.HandleFunc("/do/validateToken", validateHandler)

	// 启动服务器，监听在 8080 端口
	fmt.Println("Starting server at port 8080")
	httpErr := http.ListenAndServe(":8080", nil)
	if httpErr != nil {
		fmt.Println("Error starting server:", httpErr)
	}

}

func login(userid string) string {
	// 由于这里并没有用户表，所以不进行过滤，任何userid都可以登录

	// 创建 token
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

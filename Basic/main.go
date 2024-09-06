package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"bufio"
	"strings"
	"time"
	"strconv"
	"github.com/golang-jwt/jwt/v4"
)

// 定义与JSON结构对应的结构体
type Book struct {
	BookID          int    `json:"book_id"`
	Title           string `json:"title"`
	Author          string `json:"author"`
	PublicationDate string `json:"publication_date"`
	EntryDate       string `json:"entry_date"`
	IsBorrowed      bool   `json:"is_borrowed"`
	Borrower        string `json:"borrower"` // 使用指针以处理null值
}

type Books struct {
	Books []Book `json:"books"`
}

// Claims 结构体
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// 定义用于签名 JWT 的密钥
var jwtKey = []byte("80BEB12D58BC822705B6000584249652")

func main() {
	//定义登录信息(这里简单实现)
	username := "admin"
	password := "123456"

	// 打开JSON文件
	jsonFile, err := os.Open("Books.json")
	if err != nil {
		fmt.Println("无法打开文件", err)
		os.Exit(1)
	}
	defer jsonFile.Close()

	// 读取文件内容
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		fmt.Println("读取文件内容失败", err)
		os.Exit(1)
	}

	var booksData Books

	// 解析JSON数据
	err = json.Unmarshal(byteValue, &booksData)
	if err != nil {
		fmt.Println("序列化 JSON 时出错:", err)
		os.Exit(1)
	}
	
	var tokenString string

	//循环开始
	for {
		operationInput := input("请输入操作类别 (1：添加图书，2：删除图书，3：查询图书，4：借阅图书，5：归还图书，6：登录): ")
		//登录验证
		switch operationInput {
		case "6":
			if tokenString != "" || validateToken(tokenString) {
				fmt.Println("您已登录，无需重复登录")
				continue
			} else {
				tokenString = login(username, password)
				if tokenString == "" {
					fmt.Println("登录失败")
					continue
				} else if validateToken(tokenString) {
					fmt.Println("登录成功")
					continue
				} else {
					fmt.Println("登录发生错误，请重试")
					continue
				}
			}
		default:
			if tokenString == "" {
				fmt.Println("请先登录")
				continue
			}
			if !validateToken(tokenString) {
				fmt.Println("token已过期或无效，请重新登录")
				continue
			}

		}

		switch operationInput {
		case "1":
			//添加
			// 解析输入
			addInput := input("请输入书名、作者、出版日期（用空格分隔）")

			inputParts := strings.Split(addInput, " ")
			if len(inputParts) != 3 {
				fmt.Println("抱歉，格式错误")
				continue
			}

			title := inputParts[0]
			author := inputParts[1]
			publicationDate := inputParts[2]

			// 获取当前时间作为EntryDate
			entryDate := time.Now().Format("2006-01-02")

			// 创建新的Book实例
			newBook := Book{
				BookID:          len(booksData.Books) + 1,
				Title:           title,
				Author:          author,
				PublicationDate: publicationDate,
				EntryDate:       entryDate,
				IsBorrowed:      false,
				Borrower:        "null",
			}

			// 添加新书到Books
			booksData.Books = append(booksData.Books, newBook)

			updateJSON(booksData)

			fmt.Println("添加成功")

		case "2":
			//删除
			bookIDToDelete, _ := strconv.Atoi(input("请输入图书ID："))
			// 检查bookIDToDelete是否有效
			if bookIDToDelete < 1 || bookIDToDelete > len(booksData.Books) {
				fmt.Println("无效图书ID")
				continue
			}

			// 计算要删除的索引
			indexToDelete := bookIDToDelete - 1

			// 删除书籍
			booksData.Books = append(booksData.Books[:indexToDelete], booksData.Books[indexToDelete+1:]...)

			// 更新剩余书籍的BookID
			for i := indexToDelete; i < len(booksData.Books); i++ {
				booksData.Books[i].BookID = i + 1
			}

			updateJSON(booksData)

			fmt.Println("删除成功")
		case "3":
			//查询
			queryInput := input("请输入查询条件 (如书名：数据结构 或 作者：严蔚敏）")
			query := processQueryInput(queryInput)
			if query == nil {
				continue
			}

			isQueried := false
			switch query[0] {
			case "书名":
				for _, book := range booksData.Books {
					if book.Title == query[1] {
						fmt.Printf("图书ID: %s, 标题: %s, 作者: %s, 出版日期: %s, 是否借出: %t, 借阅者: %s\n", fmt.Sprintf("%d", book.BookID), book.Title, book.Author, book.PublicationDate, book.IsBorrowed, book.Borrower)
						isQueried = true
					}
				}
			case "作者":
				for _, book := range booksData.Books {
					if book.Author == query[1] {
						fmt.Printf("图书ID: %s, 标题: %s, 作者: %s, 出版日期: %s, 是否借出: %t, 借阅者: %s\n", fmt.Sprintf("%d", book.BookID), book.Title, book.Author, book.PublicationDate, book.IsBorrowed, book.Borrower)
						isQueried = true
					}
				}
			}

			if !isQueried {
				fmt.Println("抱歉，未查询到")
			}
		case "4":
			//借阅
			borrowInput:= input("请输入图书ID和借阅人（用空格分隔）：")
			borrowInformation := processDoubleInput(borrowInput)
			bookIDToBorrow, _ := strconv.Atoi(borrowInformation[0])
			// 检查ID是否有效
			if bookIDToBorrow < 1 || bookIDToBorrow > len(booksData.Books) {
				fmt.Println("无效图书ID")
				continue
			}

			indexToBorrow := bookIDToBorrow - 1

			if booksData.Books[indexToBorrow].IsBorrowed {
				fmt.Println("抱歉，这本书已经被借出")
				continue
			}

			// 更新书籍的借阅状态和借阅人信息
			booksData.Books[indexToBorrow].IsBorrowed = true
			booksData.Books[indexToBorrow].Borrower = borrowInformation[1]
			updateJSON(booksData)

			fmt.Println("借阅成功")

		case "5":
			//归还
			bookIDToReturn, _ := strconv.Atoi(input("请输入图书ID："))
			// 检查ID是否有效
			if bookIDToReturn < 1 || bookIDToReturn > len(booksData.Books) {
				fmt.Println("无效图书ID")
				continue
			}

			indexToReturn := bookIDToReturn - 1

			if booksData.Books[indexToReturn].IsBorrowed == false {
				fmt.Println("这本书未被借出，无需归还")
				continue
			}

			// 更新书籍的借阅状态和借阅人信息
			booksData.Books[indexToReturn].IsBorrowed = false
			booksData.Books[indexToReturn].Borrower = "null"
			updateJSON(booksData)

			fmt.Println("归还成功")
		default:
			//错误处理
			fmt.Println("请输入1-6中的数字")
		}
	}
}

func input(prompt string) string {
    reader := bufio.NewReader(os.Stdin)
    fmt.Print(prompt)
    text, _ := reader.ReadString('\n')
    return strings.TrimSpace(text)
}

func processQueryInput(input string) []string {
	parts := strings.Split(input, "：")
	if len(parts) != 2 {
		fmt.Println("抱歉，格式不符")
		return nil
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	switch key {
	case "书名":
		return []string{"书名", value}
	case "作者":
		return []string{"作者", value}
	default:
		fmt.Println("抱歉，无法查询此项")
		return nil
	}
}

func processDoubleInput(input string) []string {
	parts := strings.Split(input, " ")
	if len(parts) != 2 {
		fmt.Println("抱歉，格式不符")
		return nil
	}

	first := strings.TrimSpace(parts[0])
	second := strings.TrimSpace(parts[1])
	return []string{first, second}
}

func updateJSON(booksData Books) {
	updatedFile, err := json.MarshalIndent(booksData, "", "  ")
	if err != nil {
		fmt.Println("序列化 JSON 时出错:", err)
		return
	}

	err = ioutil.WriteFile("Books.json", updatedFile, 0644)
	if err != nil {
		fmt.Println("写入文件时出错:", err)
		return
	}
}

func login(username, password string) string {
	loginInformation := processDoubleInput(input("请输入用户名和密码（用空格分隔）："))

	if loginInformation[0] != username || loginInformation[1] != password {
		fmt.Println("用户名或密码错误")
		return ""
	}

	// 创建 token
	expirationTime := time.Now().Add(5 * time.Minute)
	claims := &Claims{
		Username: loginInformation[0],
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

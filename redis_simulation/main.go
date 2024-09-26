package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const validTime = 120

func main() {
	jsonExec := &JSONExec{"db.json"}
	data, err := jsonExec.Read()
	if err != nil {
		fmt.Println("Error while reading json file:", err)
		return
	}

	usage, err := os.Open("usage.txt")
	defer func() {
		if err := usage.Close(); err != nil {
			fmt.Println("Error while closing usage.txt:", err)
		}
	}()
	if err != nil {
		fmt.Println("Error while opening usage.txt")
		return
	}

	content, err := io.ReadAll(usage)
	if err != nil {
		fmt.Println("Error while reading usage.txt")
	}
	usageStr := string(content)

	fmt.Println("1.Enter cli\n2.Instructions for use\n3.Exit")

	startScanner := bufio.NewScanner(os.Stdin)
mainLoop:
	for startScanner.Scan() {
		initInput := startScanner.Text()

		switch initInput {
		case "1":
			fmt.Println("Welcome to cli")
			scanner := bufio.NewScanner(os.Stdin)
		cliLoop:
			for scanner.Scan() {
				input := scanner.Text()
				parts := strings.Split(input, " ")

				if len(parts) < 1 {
					fmt.Println("Invalid input")
					continue
				}

				command := strings.ToUpper(parts[0])

				switch command {
				case "SET":
					if len(parts) != 3 {
						fmt.Println("Invalid input,there should be 3 parameters,usage: SET Key Value")
						continue
					}

					timestamp := time.Now().Unix()
					index, _, _ := findValue(data, parts[1])
					if index != -1 {
						data[index].Value = parts[2]
						data[index].Timestamp = timestamp
						err = jsonExec.Update(data)
						if err != nil {
							fmt.Println("Error while updating json:", err)
						}
					} else {
						newData := Data{
							Key:       parts[1],
							Value:     parts[2],
							Timestamp: timestamp,
						}
						data = append(data, newData)

						err = jsonExec.Update(data)
						if err != nil {
							fmt.Println("Error while updating data:", err)
						} else {
							fmt.Println("Set successfully")
						}
					}

				case "SETNX":
					if len(parts) != 3 {
						fmt.Println("Invalid input,there should be 3 parameters")
						continue
					}
					timestamp := time.Now().Unix()
					index, value, _ := findValue(data, parts[1])
					if index != -1 {
						fmt.Println("0")
					} else {
						newData := Data{
							Key:       parts[1],
							Value:     value,
							Timestamp: timestamp,
						}
						data = append(data, newData)

						err = jsonExec.Update(data)
						if err != nil {
							fmt.Println("Error while updating data:", err)
						} else {
							fmt.Println("Set successfully")
						}
					}
				case "GET", "SMEMBER":
					if len(parts) != 2 {
						fmt.Println("Invalid input,there should be 2 parameters")
						continue
					}
					index, value, timestamp := findValue(data, parts[1])
					if index != -1 {
						if timestamp+validTime > time.Now().Unix() || timestamp == -1 {
							fmt.Println(value)
						} else {
							i := 0
							for _, value := range data {
								if value.Key != parts[1] {
									data[i] = value
									i++
								}
							}
							data = data[:i]

							err = jsonExec.Update(data)
							if err != nil {
								fmt.Println("Error while updating data:", err)
							} else {
								fmt.Println("Data has expired")
							}
						}
					} else {
						fmt.Println("Key not found")
					}
				case "DEL":
					if len(parts) != 2 {
						fmt.Println("Invalid input,there should be 2 parameters")
						continue
					}
					index, _, _ := findValue(data, parts[1])
					if index != -1 {
						i := 0
						for _, value := range data {
							if value.Key != parts[1] {
								data[i] = value
								i++
							}
						}

						data = data[:i]

						err = jsonExec.Update(data)
						if err != nil {
							fmt.Println("Error while updating data:", err)
						} else {
							fmt.Println("Delete successfully")
						}
					} else {
						fmt.Println("Key not found")
					}
				case "SADD":
					if len(parts) != 3 {
						fmt.Println("Invalid input,there should be 3 parameters,usage: SET Key Value")
						continue
					}

					index, valueStr, _ := findValue(data, parts[1])
					if index != -1 {
						// 这里没办法了，被转 json 搞吐了
						sliceMap := stringToSlice(valueStr, ",")
						valueSet := sliceToSet(sliceMap)
						valueSet.Add(parts[2])
						newSliceMap := setToSlice(valueSet)
						newStrMap := sliceToString(newSliceMap, ",")

						data[index].Value = newStrMap
						err = jsonExec.Update(data)
						if err != nil {
							fmt.Println("Error while updating data:", err)
						}
						fmt.Println("Set successfully")
					} else {
						valueSet := NewSet()
						valueSet.Add(parts[2])
						sliceMap := setToSlice(valueSet)
						strMap := sliceToString(sliceMap, ",")
						newData := Data{
							Key:       parts[1],
							Value:     strMap,
							Timestamp: -1,
						}
						data = append(data, newData)

						err = jsonExec.Update(data)
						if err != nil {
							fmt.Println("Error while updating data:", err)
						} else {
							fmt.Println("Set successfully")
						}
					}
				case "EXIT":
					fmt.Println("Goodbye")
					break cliLoop
				default:
					fmt.Println("Unknown command")
				}
			}
		case "2":
			fmt.Println(usageStr)
		case "3":
			fmt.Println("Bye")
			break mainLoop
		default:
			fmt.Println("Unknown command")
		}

	}
}

func findValue(data []Data, key string) (int, string, int64) {
	for index, singleData := range data {
		if singleData.Key == key {
			return index, singleData.Value, singleData.Timestamp
		}
	}
	return -1, "", 0
}

type Data struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Timestamp int64  `json:"timestamp"`
}

type JSONExec struct {
	Filename string
}

func (je *JSONExec) Read() ([]Data, error) {
	file, err := os.Open(je.Filename)
	jsonRawData, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var data []Data
	err = json.Unmarshal(jsonRawData, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (je *JSONExec) Update(data []Data) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(je.Filename, jsonData, 0644)
	if err != nil {
		return err
	} else {
		return nil
	}
}

// Set 结构体
type Set struct {
	elements map[string]struct{}
}

// NewSet 创建一个新的 Set
func NewSet() *Set {
	return &Set{
		elements: make(map[string]struct{}),
	}
}

// Add 添加元素到 Set 中
func (s *Set) Add(element string) {
	s.elements[element] = struct{}{}
}

// Remove 从 Set 中移除元素
func (s *Set) Remove(element string) {
	delete(s.elements, element)
}

func setToSlice(s *Set) []string {
	var keys []string
	for key := range s.elements {
		keys = append(keys, key)
	}
	return keys
}

// 将切片转换为 Set
func sliceToSet(slice []string) *Set {
	elements := make(map[string]struct{})
	for _, key := range slice {
		elements[key] = struct{}{}
	}
	return &Set{elements: elements}
}

// 将切片转换为字符串
func sliceToString(slice []string, delimiter string) string {
	return strings.Join(slice, delimiter)
}

// 将字符串转换为切片
func stringToSlice(str string, delimiter string) []string {
	if str == "" {
		return []string{}
	}
	return strings.Split(str, delimiter)
}

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	jsonExec := &JSONExec{"db.json"}
	data, err := jsonExec.Read()
	if err != nil {
		fmt.Println("Error while reading json file:", err)
		return
	}

	scanner := bufio.NewScanner(os.Stdin)
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

			index, _ := findValue(data, parts[1])
			if index != -1 {
				data[index].Value = parts[2]
				err = jsonExec.Update(data)
				if err != nil {
					fmt.Println("Error while updating json:", err)
				}
			} else {
				newData := Data{
					Key:   parts[1],
					Value: parts[2],
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
			index, value := findValue(data, parts[1])
			if index != -1 {
				fmt.Println("0")
			} else {
				newData := Data{
					Key:   parts[1],
					Value: value,
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
			index, value := findValue(data, parts[1])
			if index != -1 {
				fmt.Println(value)
			} else {
				fmt.Println("Key not found")
			}
		case "DEL":
			if len(parts) != 2 {
				fmt.Println("Invalid input,there should be 2 parameters")
				continue
			}
			index, _ := findValue(data, parts[1])
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

			index, valueStr := findValue(data, parts[1])
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
					Key:   parts[1],
					Value: strMap,
				}
				data = append(data, newData)

				err = jsonExec.Update(data)
				if err != nil {
					fmt.Println("Error while updating data:", err)
				} else {
					fmt.Println("Set successfully")
				}
			}
		default:
			fmt.Println("Error command")
		}
	}

}

func findValue(data []Data, key string) (int, string) {
	for index, singleData := range data {
		if singleData.Key == key {
			return index, singleData.Value
		}
	}
	return -1, ""
}

type Data struct {
	Key   string `json:"key"`
	Value string `json:"value"`
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

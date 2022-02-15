package html

import (
	"io/ioutil"
	"os"
)

// PathExists 判断文件是否存在
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// ReadFile 读取文件内容
func ReadFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return content, nil
}

// WriteFile 写入文件
func WriteFile(destFile string, content []byte) error {
	exists, _ := PathExists(destFile)
	if exists {
		// 如果文件存在，先删除
		err := os.Remove(destFile)
		if err != nil {
			return err
		}
	}
	writFile, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer writFile.Close()
	_, err = writFile.Write(content)
	if err != nil {
		return err
	}
	return nil
}

package configs

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
	"os"
)

type ConfIn interface {
	GetDirCICD() string
	GetWorkDir() string
}

type Conf struct {
	CICD string
	Work string
}

func (c *Conf) GetDirCICD() string {
	return c.CICD
}
func (c *Conf) GetWorkDir() string {
	return c.Work

}

func New(confPatch string) (*Conf, error) {
	// Получаем текущую рабочую директорию
	wd, erra := os.Getwd()
	if erra != nil {
		return nil, fmt.Errorf("ошибка получени текущего каталога: %v\n", erra)
	}

	// Формируем путь к файлу конфигурации
	configPath := wd + confPatch

	// Проверяем, существует ли файл по указанному пути
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Файл конфигурации не существует: %s\n", configPath)
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("hcl") // Указываем формат файла конфигурации

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Ошибка чтения файла конфигурации: %w", err)
	}

	// Извлекаем массив карт
	var configs []map[string]interface{}
	if err := viper.UnmarshalKey("cd", &configs); err != nil {
		return nil, fmt.Errorf("Невозможно прочитать структуру файла конфигурации: %w", err)
	}

	var (
		cicd string
		work string
	)
	// Предполагаем, что первая карта в массиве содержит наши настройки
	if len(configs) > 0 {
		cicd = configs[0]["cicd"].(string)
		work = configs[0]["work"].(string)
	} else {
		log.Fatalf("Не найдены параметры конфигурации")
	}

	return &Conf{CICD: cicd, Work: work}, nil
}

package main

import (
	"fmt"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	dirCICD string
	workDir string
)

func init() {
	logFile := &lumberjack.Logger{
		Filename:   "/var/log/info-bot/ModuleCD/ModuleCD.log",
		MaxSize:    1,    // Максимальный размер файла в мегабайтах
		MaxBackups: 5,    // Максимальное количество старых файлов для хранения
		MaxAge:     28,   // Максимальное количество дней для хранения старых файлов
		Compress:   true, // Сжимать ли старые файлы
	}
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	log.SetFlags(log.LstdFlags | log.Lshortfile) // Включаем временные метки и короткие имена файлов

	// Читаю конфигурационный файл //
	// Получаем текущую рабочую директорию
	wd, erra := os.Getwd()
	if erra != nil {
		log.Fatalf("Ошибка получени текущего каталога: %v\n", erra)
	}

	// Формируем путь к файлу конфигурации
	configPath := wd + "/CICD/cicd.conf"

	// Проверяем, существует ли файл по указанному пути
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("Файл конфигурации не существует: %s\n", configPath)
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("hcl") // Указываем формат файла конфигурации

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Ошибка чтения файла конфигурации: %w", err)
	}

	// Извлекаем массив карт
	var configs []map[string]interface{}
	if err := viper.UnmarshalKey("cd", &configs); err != nil {
		log.Fatalf("Невозможно прочитать структуру файла конфигурации: %w", err)
	}

	// Предполагаем, что первая карта в массиве содержит наши настройки
	if len(configs) > 0 {
		dirCICD = configs[0]["cicd"].(string)
		workDir = configs[0]["work"].(string)
	} else {
		log.Fatalf("Не найдены параметры конфигурации")
	}
}

func restartService(targetDir string) (string, error) {
	// Ищу файл с расширением .service что бы узнать м имя сервиса для перезапуска
	service := "none"
	files, err := os.ReadDir(targetDir)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".service" {
			service = file.Name()
		}
	}

	if service == "none" {
		log.Printf("файл .service не найден в директории %s", targetDir)
		return "", fmt.Errorf("файл .service не найден в директории %s", targetDir)
	}

	// Убираю расширение .service
	service = strings.TrimSuffix(service, ".service")

	cmd := exec.Command("systemctl", "restart", service)
	err = cmd.Run()
	if err != nil {
		log.Printf("ошибка при перезапуске сервиса %s: %w", service, err)
		return service, err
	}

	return service, nil
}

func moveNewFile(targetDir string, fileName string) (string, error) {
	//  Имя целевого файла, такое же как имя директории CI\CD
	newFileName := filepath.Base(targetDir)
	// Путь к новому файлу и старому файлу для переименования
	newPath := filepath.Join(targetDir, newFileName)
	fmt.Println("newPatch", newPath)
	oldPatch := filepath.Join(targetDir, fileName)
	fmt.Println("oldPath", oldPatch)
	// Переименовываю новый jar файл
	err := os.Rename(oldPatch, newPath)
	if err != nil {
		log.Printf("при переименовании файла %s произошла ошибка: %w", fileName, err)
		return newFileName, err
	}
	// Перемещаю файл в рабочую директорию
	srcFile, err := os.Open(newPath)
	if err != nil {
		log.Printf("при открытии исходого файла %s поризошла ошибка: %w", newPath, err)
		return newFileName, err
	}
	defer srcFile.Close()

	destFile, err := os.Create(filepath.Join(workDir, newFileName))
	if err != nil {
		log.Printf("при создании файла назначения %s произошла ошибка: %w", newFileName, err)
		return newFileName, err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		log.Printf("при копировании файла %s произошла ошибка: %w", fileName, err)
		return newFileName, err
	}
	// Закрытие всех открытых дескрипторов файла
	srcFile.Close()

	// Попытка удаления нового файла из директории CI\CD с задержкой нужна ли она?
	time.Sleep(1 * time.Second)
	err = os.Remove(newPath)
	if err != nil {
		log.Printf("ошибка при удалении файла: %w", err)
		return newFileName, err
	}

	return newFileName, nil
}

func moveOldFile(targetDir string) (string, error) {
	oldFileName := filepath.Base(targetDir)
	srcPath := filepath.Join(workDir, oldFileName)
	newFileName := oldFileName[:len(oldFileName)-len(filepath.Ext(oldFileName))] + ".old"
	destPath := filepath.Join(targetDir, newFileName)

	srcFile, err := os.Open(srcPath)
	if err != nil {
		log.Printf("не удалось открыть исходный файл: %w", err)
		return "", err
	}
	defer srcFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		log.Printf("не удалось создать файл назначения: %w", err)
		return "", err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		log.Printf("не удалось скопировать файл: %w", err)
		return "", err
	}

	// Закрытие всех открытых дескрипторов файла
	srcFile.Close()

	// Попытка удаления файла с задержкой нужна ли она?
	time.Sleep(1 * time.Second)
	err = os.Remove(srcPath)
	if err != nil {
		log.Printf("ошибка при удалении файла: %w", err)
		return "", err
	}

	return oldFileName, nil
}

func findJar(newDir os.DirEntry) {
	targetDir := filepath.Join(dirCICD, newDir.Name())

	files, err := os.ReadDir(targetDir)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		// Проверка на директорию
		if !file.IsDir() {
			// Проверка на файл с расширением .jar
			if filepath.Ext(file.Name()) == ".jar" {
				// Перемещение файла из рабочей директории в CI\CD с расширением .old
				fileName, err := moveOldFile(targetDir)
				if err != nil {
					log.Printf("При перемещении файла %s в директории %s произошла ошибка: %v\n", fileName, targetDir, err)
				} else {
					log.Printf("Файл %s в директории %s успешно перемещен с расширением .old\n", fileName, targetDir)
					// Перемещение нового файла из CI\CD в рабочую директорию с новым именем
					newFileName, err := moveNewFile(targetDir, file.Name())
					if err != nil {
						log.Printf("При перемещении нового файла %s в рабочую директории %s c произошла ошибка: %v\n", file.Name(), workDir, err)
					} else {
						log.Printf("Новый файл %s в директории %s успешно перемещен в рабочую директорию с новым именем %s\n", file.Name(), workDir, newFileName)
						// Перезапуск службы
						serviceName, err := restartService(targetDir)
						if err != nil {
							log.Printf("При перезапуске службы %s произошла ошибка: %v\n", serviceName, err)
						} else {
							log.Printf("Служба %s успешно перезапущена\n", serviceName)
						}
					}
				}
			}
		}
	}
}

func checkDir() {
	files, err := os.ReadDir(dirCICD)
	if err != nil {
		log.Fatal(err)
	}

	for _, dir := range files {
		if dir.IsDir() {
			findJar(dir)
		}
	}
}

func main() {
	for {
		checkDir()
		time.Sleep(1 * time.Minute)
	}
}

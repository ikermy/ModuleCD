package service

import (
	"ModuleCD/configs"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Service struct {
	files   []os.DirEntry
	workDir string
	cicdDir string
}

type ServiceIn interface {
	GetFiles() []os.DirEntry
}

func (s *Service) GetFiles() []os.DirEntry {
	return s.files
}

func New(conf configs.ConfIn) *Service {
	return &Service{workDir: conf.GetWorkDir(), cicdDir: conf.GetDirCICD()}
}

func (s *Service) restartService(targetDir string) (string, error) {
	// Ищу файл с расширением .service что бы узнать имя сервиса для перезапуска
	service := "none"
	files, err := os.ReadDir(targetDir)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".service" {
			service = file.Name()
		}
	}

	if service == "none" {
		return "", fmt.Errorf("файл .service не найден в директории %s", targetDir)
	}

	// Убираю расширение .service
	service = strings.TrimSuffix(service, ".service")

	cmd := exec.Command("systemctl", "restart", service)
	err = cmd.Run()
	if err != nil {
		return service, fmt.Errorf("ошибка при перезапуске сервиса %s: %w", service, err)
	}

	return service, nil
}

func (s *Service) DoIt(tardetDir string, file os.DirEntry) (string, string, string, error) {
	// Перемещение файла из рабочей директории в CI\CD с расширением .old
	oldFileName, err := s.moveOldFile(tardetDir)
	if err != nil {
		return "", "", "", fmt.Errorf("при перемещении файла %s в директории %s произошла ошибка: %w", oldFileName, tardetDir, err)
	}
	// Перемещение нового файла из CI\CD в рабочую директорию с новым именем
	newFileName, err := s.moveNewFile(tardetDir, file)
	if err != nil {
		return "", "", "", fmt.Errorf("при перемещении нового файла %s в рабочую директории %s c произошла ошибка: %w", file, s.workDir, err)
	}

	// Перезапуск сервиса
	service, err := s.restartService(tardetDir)
	if err != nil {
		return oldFileName, newFileName, "", fmt.Errorf("при перезапуске сервиса произошла ошибка: %w", err)
	}

	return oldFileName, newFileName, service, nil
}

func (s *Service) moveNewFile(targetDir string, file os.DirEntry) (string, error) {
	//  Имя целевого файла, такое же как имя директории CI\CD
	newFileName := filepath.Base(targetDir)
	// Путь к новому файлу и старому файлу для переименования
	newPath := filepath.Join(targetDir, newFileName)
	oldPatch := filepath.Join(targetDir, file.Name())
	// Переименовываю новый jar файл
	err := os.Rename(oldPatch, newPath)
	if err != nil {
		return newFileName, fmt.Errorf("при переименовании файла %s произошла ошибка: %w", file, err)
	}
	// Перемещаю файл в рабочую директорию
	srcFile, err := os.Open(newPath)
	if err != nil {
		return newFileName, fmt.Errorf("при открытии исходого файла %s поризошла ошибка: %w", newPath, err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(filepath.Join(s.workDir, newFileName))
	if err != nil {
		return newFileName, fmt.Errorf("при создании файла назначения %s произошла ошибка: %w", newFileName, err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return newFileName, fmt.Errorf("при копировании файла %s произошла ошибка: %w", file, err)
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

func (s *Service) moveOldFile(targetDir string) (string, error) {
	oldFileName := filepath.Base(targetDir)
	srcPath := filepath.Join(s.workDir, oldFileName)
	newFileName := oldFileName[:len(oldFileName)-len(filepath.Ext(oldFileName))] + ".old"
	destPath := filepath.Join(targetDir, newFileName)

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("не удалось открыть исходный файл: %w", err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("не удалось создать файл назначения: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return "", fmt.Errorf("не удалось скопировать файл: %w", err)
	}

	// Закрытие всех открытых дескрипторов файла
	srcFile.Close()

	// Попытка удаления файла с задержкой нужна ли она?
	time.Sleep(1 * time.Second)
	err = os.Remove(srcPath)
	if err != nil {
		return "", fmt.Errorf("ошибка при удалении файла: %w", err)
	}

	return oldFileName, nil
}

func (s *Service) FindJar(newDir os.DirEntry) (string, os.DirEntry, error) {
	targetDir := filepath.Join(s.cicdDir, newDir.Name())

	files, err := os.ReadDir(targetDir)
	if err != nil {
		return "", nil, err
	}

	for _, file := range files {
		// Проверка на директорию
		if !file.IsDir() {
			// Проверка на файл с расширением .jar
			if filepath.Ext(file.Name()) == ".jar" {
				return targetDir, file, nil
			}
		}
	}
	// Нет файла с расширением .jar
	return targetDir, nil, nil
}

func (s *Service) CheckDir() error {
	{
		files, err := os.ReadDir(s.cicdDir)
		if err != nil {
			return err
		}
		s.files = files
		return nil
	}
}

package endpoint

import (
	"log"
	"os"
	"time"
)

type Endpoint struct {
	s Service
}

type Service interface {
	CheckDir() error
	FindJar(newDir os.DirEntry) (string, os.DirEntry, error)
	GetFiles() []os.DirEntry
	DoIt(targetDir string, file os.DirEntry) (string, string, string, error)
}

func New(s Service) *Endpoint {
	return &Endpoint{s: s}
}

func (e *Endpoint) Work() error {
	for {
		for _, dir := range e.s.GetFiles() {
			if dir.IsDir() {
				targetDir, file, err := e.s.FindJar(dir)
				if err != nil {
					return err
				}
				if file != nil {
					oldFileName, newFileName, service, err := e.s.DoIt(targetDir, file)
					if err != nil {
						log.Printf("Внимание %v\n", err)
					} else {
						log.Printf("Старый файл: %s Новый файл: %s\n", oldFileName, newFileName)
						log.Printf("Служба перезапущена: %s\n", service)
					}
				}
			}
		}
		time.Sleep(1 * time.Minute)
	}
}

func (e *Endpoint) Start() error {
	log.Print("ModuleCD запускается...")

	err := e.s.CheckDir()
	if err != nil {
		return err
	}

	err = e.Work()
	if err != nil {
		return err
	}

	return nil
}

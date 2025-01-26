package app

import (
	"ModuleCD/configs"
	"ModuleCD/internal/app/endpoint"
	"ModuleCD/internal/app/service"
)

type App struct {
	e *endpoint.Endpoint
	s *service.Service
}

func New(conf *configs.Conf) *App {
	a := &App{}
	a.s = service.New(conf)
	a.e = endpoint.New(a.s)

	return a
}

func (a *App) Run() error {
	err := a.e.Start()
	if err != nil {
		return err
	}

	return nil
}

package handler

import (
	"fmt"
	"github.com/Miyagawa-Ryohei/mkmicro/entity"
)

type SampleHandler struct {}

func (h SampleHandler) Exec(msg entity.Message, dist entity.SessionManager) bool {
	fmt.Printf("%s",msg.GetBody())
	return true
}

func (h SampleHandler) GetResultQueueConfig() *entity.QueueConfig {
	return nil
}

func (h SampleHandler) GetResultSessionConfig() *entity.SessionConfig {
	return nil
}
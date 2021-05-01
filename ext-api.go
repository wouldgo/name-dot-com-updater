package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/namedotcom/go/namecom"
)

func ipify() (string, error) {
	resp, err := http.Get("https://api.ipify.org/?format=json")

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return "", readErr
	}

	type ipifyS struct {
		Ip string `json:"ip"`
	}

	Ipify := ipifyS{}
	if err := json.Unmarshal(body, &Ipify); err != nil {
		return "", fmt.Errorf("error decoding solver config: %v", err)
	}

	return Ipify.Ip, nil
}

func New() (EnvironmentConf, error) {
	type nameDotComConf struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var LogLevel = os.Getenv("LOG_LEVEL")
	var Domains = os.Getenv("DOMAINS")
	var NameDotComConfs = os.Getenv("NAME_DOT_COM_CONFS")

	if LogLevel == "" {
		LogLevel = "DEBUG"
	}
	if Domains == "" {
		return EnvironmentConf{}, fmt.Errorf("DOMAINS cannot be empty")
	}
	if NameDotComConfs == "" {
		return EnvironmentConf{}, fmt.Errorf("NAME_DOT_COM_CONFS cannot be empty")
	}

	DomainsList := strings.Split(Domains, ",")

	NameDotComConfsStruct := nameDotComConf{}

	if err := json.Unmarshal([]byte(NameDotComConfs), &NameDotComConfsStruct); err != nil {
		return EnvironmentConf{}, fmt.Errorf("error decoding solver config: %v", err)
	}

	return EnvironmentConf{
		LogLevel:         LogLevel,
		Domains:          DomainsList,
		NameDotComClient: namecom.New(NameDotComConfsStruct.Username, NameDotComConfsStruct.Password),
	}, nil
}

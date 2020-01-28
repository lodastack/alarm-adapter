package models

import (
	"time"
)

const timeFormat = "2006-01-02 15:04:05"

type Report struct {
	UUID        []string  `json:"uuid"`
	SN          string    `json:"sn"`
	NewIPList   []string  `json:"newiplist"`
	OldIPList   []string  `json:"oldiplist"`
	Ns          []string  `json:"ns"`
	Version     string    `json:"version"`
	Commit      string    `json:"commit"`
	Branch      string    `json:"branch"`
	BuildTime   string    `json:"buildtime"`
	GoVersion   string    `json:"goversion"`
	NewHostname string    `json:"newhostname"`
	OldHostname string    `json:"oldhostname"`
	AgentType   string    `json:"agenttype"`
	Update      bool      `json:"update"`
	UpdateTime  time.Time `json:"utime"`
}

func (r *Report) Marshal() map[string]string {
	return map[string]string{
		"version":    r.Version,
		"commit":     r.Commit,
		"branch":     r.Branch,
		"buildTime":  r.BuildTime,
		"goVersion":  r.GoVersion,
		"agentType":  r.AgentType,
		"lastReport": r.UpdateTime.Format(timeFormat),
	}
}

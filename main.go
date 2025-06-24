package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
	"time"
)

const statsFile = "/run/hive/last_stat.json"

//const statsFile = "stat.json" //dev

const errSleepInterval = 5
const successSleepInterval = 5

type StatsStruct struct {
	Method string `json:"method"`
	Params struct {
		V      int    `json:"v"`
		RigID  string `json:"rig_id"`
		Passwd string `json:"passwd"`
		Meta   struct {
			FsID   int `json:"fs_id"`
			Custom struct {
				Coin string `json:"coin"`
			} `json:"custom"`
		} `json:"meta"`
		Temp       []int     `json:"temp"`
		MTemp      []int     `json:"mtemp"`
		JTemp      []int     `json:"jtemp"`
		Fan        []int     `json:"fan"`
		Power      []int     `json:"power"`
		Df         string    `json:"df"`
		Mem        []int     `json:"mem"`
		Cputemp    []int     `json:"cputemp"`
		Cpuavg     []float64 `json:"cpuavg"`
		Jtemp      []int     `json:"jtemp"`
		Miner      string    `json:"miner"`
		TotalKhs   float64   `json:"total_khs"`
		MinerStats struct {
			Status  string    `json:"status"`
			Khs     []float64 `json:"khs"`
			HsUnits string    `json:"hs_units"`
			Ver     string    `json:"ver"`
			Algo    string    `json:"algo"`
		} `json:"miner_stats"`
		MknetAutofanStats struct {
			Casefan       []int         `json:"casefan"`
			Thermosensors []interface{} `json:"thermosensors"`
		} `json:"mknet_autofan_stats"`
	} `json:"params"`
}

func errSleep(err error) {
	log.Println(err)
	log.Println("Sleeping 5s...")
	time.Sleep(time.Second * errSleepInterval)
}

func recordMetrics(gauges map[string]*prometheus.GaugeVec) {
	var (
		bytes []byte
		stats StatsStruct
		err   error
	)

	for {
		for {
			if _, err := os.Stat(statsFile); errors.Is(err, os.ErrNotExist) {
				errSleep(err)
			} else {
				break
			}
		}

		bytes, err = os.ReadFile(statsFile)
		if err != nil {
			errSleep(err)
			continue
		}

		//fmt.Printf("%s", bytes)

		err = json.Unmarshal(bytes, &stats)
		if err != nil {
			_ = err
			//fmt.Println(err)
		}

		hash := stats.Params.MinerStats.Khs
		cTemps := stats.Params.Temp
		mTemps := stats.Params.MTemp
		jTemps := stats.Params.JTemp
		power := stats.Params.Power
		fan := stats.Params.Fan
		caseFan := stats.Params.MknetAutofanStats.Casefan
		totalHash := stats.Params.TotalKhs * 1e3

		rigId := stats.Params.RigID

		for i, h := range hash {
			gauges["hash"].With(prometheus.Labels{"rig": rigId, "card": fmt.Sprintf("%d", i)}).Set(h)
		}
		gauges["hash"].With(prometheus.Labels{"rig": rigId, "card": "total"}).Set(totalHash)

		for i := range cTemps {
			gauges["core_temp"].With(prometheus.Labels{"rig": rigId, "card": fmt.Sprintf("%d", i)}).Set(float64(cTemps[i]))
			if len(mTemps) > 0 {
				gauges["mem_temp"].With(prometheus.Labels{"rig": rigId, "card": fmt.Sprintf("%d", i)}).Set(float64(mTemps[i]))
			}
			if len(jTemps) > 0 {
				gauges["junc_temp"].With(prometheus.Labels{"rig": rigId, "card": fmt.Sprintf("%d", i)}).Set(float64(jTemps[i]))
			}
			gauges["power"].With(prometheus.Labels{"rig": rigId, "card": fmt.Sprintf("%d", i)}).Set(float64(power[i]))
			gauges["fan"].With(prometheus.Labels{"rig": rigId, "card": fmt.Sprintf("%d", i)}).Set(float64(fan[i]))
		}

		for i := range caseFan {
			gauges["case_fan"].With(prometheus.Labels{"rig": rigId, "fan": fmt.Sprintf("%d", i)}).Set(float64(caseFan[i]))
		}

		time.Sleep(time.Second * successSleepInterval)
	}
}

func main() {

	gauges := map[string]*prometheus.GaugeVec{
		"hash":      prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "hive_hashrate", Help: "Hashrate"}, []string{"rig", "card"}),
		"core_temp": prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "hive_coretemp", Help: "GPU Core Temp"}, []string{"rig", "card"}),
		"mem_temp":  prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "hive_memtemp", Help: "GPU Memory Temperature"}, []string{"rig", "card"}),
		"junc_temp": prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "hive_junctemp", Help: "GPU Junction Temperature"}, []string{"rig", "card"}),
		"power":     prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "hive_power", Help: "GPU Power Consumption"}, []string{"rig", "card"}),
		"fan":       prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "hive_fan", Help: "GPU Fan Speed"}, []string{"rig", "card"}),
		"case_fan":  prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "hive_casefan", Help: "Case Fan Speed"}, []string{"rig", "fan"}),
	}
	promReg := prometheus.NewRegistry()
	for _, g := range gauges {
		promReg.MustRegister(g)
	}

	go recordMetrics(gauges)

	handler := promhttp.HandlerFor(
		promReg,
		promhttp.HandlerOpts{EnableOpenMetrics: false},
	)

	http.Handle("/metrics", handler)
	http.ListenAndServe(":2112", nil)

}

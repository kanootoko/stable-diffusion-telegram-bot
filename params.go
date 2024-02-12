package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
)

type paramsType struct {
	BotToken               string
	StableDiffusionApiHost string

	AllowedUserIDs  []int64
	AdminUserIDs    []int64
	AllowedGroupIDs []int64

	DefaultModel      string
	DefaultSampler    string
	DefaultCnt        int
	DefaultBatch      int
	DefaultSteps      int
	DefaultWidth      int
	DefaultHeight     int
	DefaultWidthSDXL  int
	DefaultHeightSDXL int
	DefaultStepsSDXL  int
	DefaultCFGScale   float64
}

type defaultsDefaults struct {
	StableDiffusionApiHost string
	Model                  string
	Sampler                string
	Cnt                    int
	Batch                  int
	Steps                  int
	Width                  int
	Height                 int
	WidthSDXL              int
	HeightSDXL             int
	StepsSDXL              int
	CFGScale               float64
}

func getDefaultsFromEnv() (defaults defaultsDefaults) {
	if value, isSet := os.LookupEnv("STABLE_DIFFUSION_API"); isSet {
		defaults.StableDiffusionApiHost = value
	} else {
		defaults.StableDiffusionApiHost = "http://localhost:7860"
	}

	if value, isSet := os.LookupEnv("DEFAULT_MODEL"); isSet {
		defaults.Model = value
	}

	if value, isSet := os.LookupEnv("DEFAULT_SAMPLER"); isSet {
		defaults.Sampler = value
	}

	if value, isSet := os.LookupEnv("DEFAULT_WIDTH"); isSet {
		if intValue, err := strconv.Atoi(value); err == nil {
			defaults.Width = intValue
		} else {
			defaults.Width = 512
		}
	} else {
		defaults.Width = 512
	}

	if value, isSet := os.LookupEnv("DEFAULT_HEIGHT"); isSet {
		if intValue, err := strconv.Atoi(value); err == nil {
			defaults.Height = intValue
		} else {
			defaults.Height = 512
		}
	} else {
		defaults.Height = 512
	}

	if value, isSet := os.LookupEnv("DEFAULT_STEPS"); isSet {
		if intValue, err := strconv.Atoi(value); err == nil {
			defaults.Steps = intValue
		} else {
			defaults.Steps = 30
		}
	} else {
		defaults.Steps = 30
	}

	if value, isSet := os.LookupEnv("DEFAULT_CNT"); isSet {
		if intValue, err := strconv.Atoi(value); err == nil {
			defaults.Cnt = intValue
		} else {
			defaults.Cnt = 2
		}
	} else {
		defaults.Cnt = 2
	}

	if value, isSet := os.LookupEnv("DEFAULT_BATCH"); isSet {
		if intValue, err := strconv.Atoi(value); err == nil {
			defaults.Batch = intValue
		} else {
			defaults.Batch = 1
		}
	} else {
		defaults.Batch = 1
	}

	if value, isSet := os.LookupEnv("DEFAULT_WIDTH_SDXL"); isSet {
		if intValue, err := strconv.Atoi(value); err == nil {
			defaults.WidthSDXL = intValue
		} else {
			defaults.WidthSDXL = 512
		}
	} else {
		defaults.WidthSDXL = 512
	}

	if value, isSet := os.LookupEnv("DEFAULT_HEIGHT_SDXL"); isSet {
		if intValue, err := strconv.Atoi(value); err == nil {
			defaults.HeightSDXL = intValue
		} else {
			defaults.HeightSDXL = 512
		}
	} else {
		defaults.HeightSDXL = 512
	}

	if value, isSet := os.LookupEnv("DEFAULT_STEPS_SDXL"); isSet {
		if intValue, err := strconv.Atoi(value); err == nil {
			defaults.StepsSDXL = intValue
		} else {
			defaults.StepsSDXL = 25
		}
	} else {
		defaults.StepsSDXL = 25
	}
	if value, isSet := os.LookupEnv("DEFAULT_CFG_SCALE"); isSet {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			defaults.CFGScale = floatValue
		} else {
			defaults.CFGScale = 7.0
		}
	} else {
		defaults.CFGScale = 7.0
	}
	return
}

func (p *paramsType) Init() error {

	defaults := getDefaultsFromEnv()

	flag.StringVar(&p.BotToken, "bot-token", "", "telegram bot token [required]")
	flag.StringVar(&p.StableDiffusionApiHost, "sd-api", defaults.StableDiffusionApiHost, "address of running Stable Diffusion AUTOMATIC1111 API")
	var allowedUserIDs string
	flag.StringVar(&allowedUserIDs, "allowed-user-ids", "", "allowed telegram user ids")
	var adminUserIDs string
	flag.StringVar(&adminUserIDs, "admin-user-ids", "", "admin telegram user ids")
	var allowedGroupIDs string
	flag.StringVar(&allowedGroupIDs, "allowed-group-ids", "", "allowed telegram group ids")
	flag.StringVar(&p.DefaultModel, "default-model", defaults.Model, "default model name")
	flag.StringVar(&p.DefaultSampler, "default-sampler", defaults.Sampler, "default sampler name")
	flag.IntVar(&p.DefaultCnt, "default-cnt", defaults.Cnt, "default images count")
	flag.IntVar(&p.DefaultBatch, "default-batch", defaults.Batch, "default images batch size")
	flag.IntVar(&p.DefaultSteps, "default-steps", defaults.Steps, "default generation steps")
	flag.IntVar(&p.DefaultWidth, "default-width", defaults.Width, "default image width")
	flag.IntVar(&p.DefaultHeight, "default-height", defaults.Height, "default image height")
	flag.IntVar(&p.DefaultWidthSDXL, "default-width-sdxl", defaults.WidthSDXL, "default image width for SDXL models")
	flag.IntVar(&p.DefaultHeightSDXL, "default-height-sdxl", defaults.HeightSDXL, "default image height for SDXL models")
	flag.IntVar(&p.DefaultStepsSDXL, "default-cnt-sdxl", defaults.StepsSDXL, "default generation steps count for SDXL models")
	flag.Float64Var(&p.DefaultCFGScale, "default-cfg-scale", defaults.CFGScale, "default CFG scale")
	flag.Parse()

	if p.BotToken == "" {
		p.BotToken = os.Getenv("BOT_TOKEN")
	}
	if p.BotToken == "" {
		return fmt.Errorf("bot token not set")
	}

	if allowedUserIDs == "" {
		allowedUserIDs = os.Getenv("ALLOWED_USERIDS")
	}
	sa := strings.Split(allowedUserIDs, ",")
	for _, idStr := range sa {
		if idStr == "" {
			continue
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return fmt.Errorf("allowed user ids contains invalid user ID: " + idStr)
		}
		p.AllowedUserIDs = append(p.AllowedUserIDs, id)
	}

	if adminUserIDs == "" {
		adminUserIDs = os.Getenv("ADMIN_USERIDS")
	}
	sa = strings.Split(adminUserIDs, ",")
	for _, idStr := range sa {
		if idStr == "" {
			continue
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return fmt.Errorf("admin ids contains invalid user ID: " + idStr)
		}
		p.AdminUserIDs = append(p.AdminUserIDs, id)
		if !slices.Contains(p.AllowedUserIDs, id) {
			p.AllowedUserIDs = append(p.AllowedUserIDs, id)
		}
	}

	if allowedGroupIDs == "" {
		allowedGroupIDs = os.Getenv("ALLOWED_GROUPIDS")
	}
	sa = strings.Split(allowedGroupIDs, ",")
	for _, idStr := range sa {
		if idStr == "" {
			continue
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return fmt.Errorf("allowed group ids contains invalid group ID: " + idStr)
		}
		p.AllowedGroupIDs = append(p.AllowedGroupIDs, id)
	}

	return nil
}

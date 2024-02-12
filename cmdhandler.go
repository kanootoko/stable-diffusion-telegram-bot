package main

import (
	"context"
	"fmt"
	"math/rand"
	"os/exec"
	"strings"

	"github.com/go-telegram/bot/models"
)

type cmdHandlerType struct {
	sdApi  *sdAPIType
	params DefaultGenerationParameters
}

func (c *cmdHandlerType) SD(ctx context.Context, msg *models.Message) {
	reqParams := ReqParamsRender{
		origPrompt:  msg.Text,
		Seed:        rand.Uint32(),
		Width:       c.params.DefaultWidth,
		Height:      c.params.DefaultHeight,
		Steps:       c.params.DefaultSteps,
		NumOutputs:  c.params.DefaultCnt,
		CFGScale:    c.params.DefaultCFGScale,
		SamplerName: c.params.DefaultSampler,
		ModelName:   c.params.DefaultModel,
		Upscale: ReqParamsUpscale{
			Upscaler: "LDSR",
		},
		HR: ReqParamsRenderHR{
			DenoisingStrength: 0.4,
			Upscaler:          "R-ESRGAN 4x+",
			SecondPassSteps:   15,
		},
	}

	var paramsLine *string
	lines := strings.Split(msg.Text, "\n")
	if len(lines) >= 2 {
		reqParams.Prompt = lines[0]
		reqParams.NegativePrompt = strings.Join(lines[1:], " ")
		paramsLine = &reqParams.NegativePrompt
	} else {
		reqParams.Prompt = msg.Text
		paramsLine = &reqParams.Prompt
	}
	firstCmdCharAt, err := ReqParamsParse(ctx, c.sdApi, c.params, *paramsLine, &reqParams)
	if err != nil {
		sendReplyToMessage(ctx, msg, errorStr+": can't parse render params: "+err.Error())
		return
	}
	if firstCmdCharAt >= 0 { // Commands found? Removing them from the line.
		*paramsLine = (*paramsLine)[:firstCmdCharAt]
	}

	reqParams.Prompt = strings.TrimSpace(reqParams.Prompt)
	reqParams.NegativePrompt = strings.TrimSpace(reqParams.NegativePrompt)

	if reqParams.Prompt == "" {
		fmt.Println("  missing prompt")
		sendReplyToMessage(ctx, msg, errorStr+": missing prompt")
		return
	}

	if reqParams.HR.Scale > 0 || reqParams.Upscale.Scale > 0 {
		reqParams.NumOutputs = 1
	}

	req := ReqQueueReq{
		Type:    ReqTypeRender,
		Message: msg,
		Params:  reqParams,
	}
	reqQueue.Add(req)
}

func (c *cmdHandlerType) Upscale(ctx context.Context, msg *models.Message) {
	reqParams := ReqParamsUpscale{
		origPrompt: msg.Text,
		Scale:      4,
		Upscaler:   "LDSR",
	}

	_, err := ReqParamsParse(ctx, c.sdApi, c.params, msg.Text, &reqParams)
	if err != nil {
		sendReplyToMessage(ctx, msg, errorStr+": can't parse render params: "+err.Error())
		return
	}

	req := ReqQueueReq{
		Type:    ReqTypeUpscale,
		Message: msg,
		Params:  reqParams,
	}
	reqQueue.Add(req)
}

func (c *cmdHandlerType) Cancel(ctx context.Context, msg *models.Message) {
	if err := reqQueue.CancelCurrentEntry(ctx); err != nil {
		sendReplyToMessage(ctx, msg, errorStr+": "+err.Error())
	}
}

func (c *cmdHandlerType) Models(ctx context.Context, msg *models.Message) {
	models, err := c.sdApi.GetModels(ctx)
	if err != nil {
		fmt.Println("  error getting models:", err)
		sendReplyToMessage(ctx, msg, errorStr+": error getting models: "+err.Error())
		return
	}
	for i := range models {
		if models[i] == c.params.DefaultModel {
			models[i] = "- <b>" + models[i] + "</b> (default)"
		} else {
			models[i] = "- <code>" + models[i] + "</code>"
		}
	}
	res := strings.Join(models, "\n")
	var text string
	if res != "" {
		text = "🧩 Available models:\n" + res
	} else {
		text = "No available models."
	}
	sendReplyToMessage(ctx, msg, text)
}

func (c *cmdHandlerType) Samplers(ctx context.Context, msg *models.Message) {
	samplers, err := c.sdApi.GetSamplers(ctx)
	if err != nil {
		fmt.Println("  error getting samplers:", err)
		sendReplyToMessage(ctx, msg, errorStr+": error getting samplers: "+err.Error())
		return
	}
	for i := range samplers {
		if samplers[i] == c.params.DefaultSampler {
			samplers[i] = "- <b>" + samplers[i] + "</b> (default)"
		} else {
			samplers[i] = "- <code>" + samplers[i] + "</code>"
		}
	}
	res := strings.Join(samplers, ", ")
	var text string
	if res != "" {
		text = "🔭 Available samplers:\n" + res
	} else {
		text = "No available samplers."
	}
	sendReplyToMessage(ctx, msg, text)
}

func (c *cmdHandlerType) Embeddings(ctx context.Context, msg *models.Message) {
	embs, err := c.sdApi.GetEmbeddings(ctx)
	if err != nil {
		fmt.Println("  error getting embeddings:", err)
		sendReplyToMessage(ctx, msg, errorStr+": error getting embeddings: "+err.Error())
		return
	}
	for i := range embs {
		embs[i] = "- <code>" + embs[i] + "</code>"
	}
	res := strings.Join(embs, ", ")
	var text string
	if res != "" {
		text = "Available embeddings: " + res
	} else {
		text = "No available embeddings."
	}
	sendReplyToMessage(ctx, msg, text)
}

func (c *cmdHandlerType) LoRAs(ctx context.Context, msg *models.Message) {
	loras, err := c.sdApi.GetLoRAs(ctx)
	if err != nil {
		fmt.Println("  error getting loras:", err)
		sendReplyToMessage(ctx, msg, errorStr+": error getting loras: "+err.Error())
		return
	}
	for i := range loras {
		loras[i] = "- <code>" + loras[i] + "</code>"
	}
	res := strings.Join(loras, ", ")
	var text string
	if res != "" {
		text = "Available LoRAs: " + res
	} else {
		text = "No available LoRAs."
	}
	sendReplyToMessage(ctx, msg, text)
}

func (c *cmdHandlerType) Upscalers(ctx context.Context, msg *models.Message) {
	ups, err := c.sdApi.GetUpscalers(ctx)
	if err != nil {
		fmt.Println("  error getting upscalers:", err)
		sendReplyToMessage(ctx, msg, errorStr+": error getting upscalers: "+err.Error())
		return
	}
	for i := range ups {
		ups[i] = "- <code>" + ups[i] + "</code>"
	}
	res := strings.Join(ups, ", ")
	var text string
	if res != "" {
		text = "🔎 Available upscalers: " + res
	} else {
		text = "🔎 No available upscalers."
	}
	sendReplyToMessage(ctx, msg, text)
}

func (c *cmdHandlerType) VAEs(ctx context.Context, msg *models.Message) {
	vaes, err := c.sdApi.GetVAEs(ctx)
	if err != nil {
		fmt.Println("  error getting vaes:", err)
		sendReplyToMessage(ctx, msg, errorStr+": error getting vaes: "+err.Error())
		return
	}
	for i := range vaes {
		vaes[i] = "- <code>" + vaes[i] + "</code>"
	}
	res := strings.Join(vaes, ", ")
	var text string
	if res != "" {
		text = "Available VAEs: " + res
	} else {
		text = "No available VAEs."
	}
	sendReplyToMessage(ctx, msg, text)
}

func (c *cmdHandlerType) SMI(ctx context.Context, msg *models.Message) {
	cmd := exec.Command("nvidia-smi")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("  error running nvidia-smi:", err)
		sendReplyToMessage(ctx, msg, errorStr+": error running nvidia-smi: "+err.Error())
		return
	}
	sendReplyToMessage(ctx, msg, "<pre>"+string(out)+"</pre>")
}

func (c *cmdHandlerType) Help(ctx context.Context, msg *models.Message, cmdChar string) {
	sendReplyToMessage(
		ctx,
		msg,
		"🤖 Stable Diffusion Telegram Bot\n\n"+
			"Available commands:\n\n"+

			cmdChar+"sd [prompt] - render prompt (negative prompt can be put"+
			" on the next line)\n"+
			cmdChar+"upscale - upscale image\n"+
			cmdChar+"cancel - cancel ongoing request\n"+
			cmdChar+"models - list available models\n"+
			cmdChar+"samplers - list available samplers\n"+
			cmdChar+"embeddings - list available embeddings\n"+
			cmdChar+"loras - list available LoRAs\n"+
			cmdChar+"upscalers - list available upscalers\n"+
			cmdChar+"vaes - list available VAEs\n"+
			cmdChar+"smi - get the output of nvidia-smi\n"+
			cmdChar+"help - show this help\n\n"+

			"Available render parameters at the end of the prompt:\n\n"+

			"-seed/s - set seed\n"+
			"-width/w - set output image width\n"+
			"-height/h - set output image height\n"+
			"-steps/t - set the number of steps\n"+
			"-cnt/o - set count of output images\n"+
			"-batch/b - set batch size of output images\n"+
			"-png - upload PNGs instead of JPEGs\n"+
			"-cfg/c - set CFG scale\n"+
			"-sampler/r - set sampler, get valid values with /samplers\n"+
			"-model/m - set model, get valid values with /models\n"+
			"-upscale/u - upscale output image with ratio\n"+
			"-upscaler - set upscaler method, get valid values with /upscalers\n"+
			"-hr - enable highres mode and set upscale ratio\n"+
			"-hr-denoisestrength/hrd - set highres mode denoise strength\n"+
			"-hr-upscaler/hru - set highres mode upscaler, get valid values with /upscalers\n"+
			"-hr-steps/hrt - set the number of highres mode second pass steps\n\n"+

			"Available upscale parameters:\n\n"+

			"-upscale/u - upscale output image with ratio\n"+
			"-upscaler - set upscaler method, get valid values with /upscalers\n"+
			"-png - upload PNGs instead of JPEGs\n\n"+

			"For more information see https://github.com/kanootoko/stable-diffusion-telegram-bot",
	)
}

package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
)

const (
	StreamPrefix  = "data:"
	StreamEOF     = "EOF"
	StreamSuccess = "success"
	letterBytes   = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	SSEServerIP   = "http://localhost"
	SSEServerPost = "8844"
	SSEServerPath = "botlite/api/v2/stream"
)

type Input struct {
	BizID       uint64       `json:"biz_id" doc:"业务ID"`
	SessionID   string       `json:"session_id" doc:"会话ID"`
	Query       string       `json:"query" doc:"问句"`
	NeedDump    bool         `json:"need_dump" doc:"会话是否需要持久化，默认持久化" spec:"optional,default=true"`
	EmotionOn   bool         `json:"emotion_on" doc:"是否开启情感识别" spec:"optional,default=false"`
	UserID      string       `json:"user_id" doc:"用户ID" spec:"optional,default="`
	Client      string       `json:"client" doc:"渠道" spec:"optional,default="`
	DriveParams []DriveParam `json:"drive_params" doc:"随路参数列表"`
	ExtraParams []TypedParam `json:"extra_params" doc:"随路参数列表" spec:"optional"`
}

type DriveParam struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

type TypedParam struct {
	Type   string  `json:"type" doc:"参数类型，目前支持类型slot，用于传输不依赖ner的随路词槽（如正在询问商品）"`
	Params []Param `json:"params" doc:"参数列表"`
}

type Param struct {
	Key   string `json:"key" doc:"参数名"`
	Value string `json:"value" doc:"参数值"`
}

type QueryRspV2 struct {
	BizID         uint64        `json:"biz_id" doc:"业务ID"`
	SessionID     string        `json:"session_id" doc:"会话ID"`
	SearchID      string        `json:"search_id" doc:"本次请求的search_id"`
	Query         string        `json:"query" doc:"问句"`
	AnswerList    []Answer      `json:"answer_list" doc:"答案，可能是双/多答案"`
	IntentMatches []Intent      `json:"intent_matches" doc:"意图命中情况"`
	TaskStatus    TaskStatus    `json:"task_status" doc:"任务状态"`
	IsRejected    bool          `json:"is_rejected" doc:"是否拒识"`
	Sentiments    []Sentiment   `json:"sentiment" doc:"情感识别结果"`
	Sensitivity   Sensitivity   `json:"sensitivity" doc:"敏感识别结果"`
	Slots         []Slot        `json:"slots" doc:"词槽识别结果"`
	Actions       []EventAction `json:"actions" doc:"命中的事件对应的动作列表"`
	TimeCost      string        `json:"time_cost" doc:"所有模块花费的时间"`
	Round         int           `json:"round" doc:"轮次"`
	Msg           string        `json:"msg" doc:"非拒识时为success，拒识时可能附带错误信息，EOF代表一轮流式传输结束"`
}

type Answer struct {
	Type            string   `json:"type" doc:"回答的bot类型" doc:"faq, task, chat, dm"`
	BizIntentID     uint64   `json:"biz_intent_id" doc:"业务意图ID"`
	Text            string   `json:"text" doc:"文本答案"`
	RelateQuestions []string `json:"relate_questions" doc:"关联问列表"`
}

type Intent struct {
	BizIntentID uint64  `json:"biz_intent_id" doc:"业务意图ID"`
	Name        string  `json:"name" doc:"标准问"`
	Type        string  `json:"type" doc:"意图类型，FAQ，TASK，CHAT"`
	ExactHit    bool    `json:"exact_hit" doc:"是否精准命中"`
	Rank        int     `json:"rank" doc:"意图排序，可能经过重排序，与相似度打分不匹配"`
	Score       float32 `json:"score" doc:"相似度打分，最高1.0"`
	IntentID    uint64  `json:"intent_id" doc:"公共库意图ID，如果为0则是自定义意图"`
}

type TaskStatus struct {
	ID           uint64          `json:"id" doc:"业务任务ID"`
	TaskID       uint64          `json:"task_id" doc:"公共库task_id，当为自定义任务时，task_id=0"`
	Type         string          `json:"type" doc:"任务类别，表示哪种类型的任务子bot作了回答，举例：dialflow, skill, slotfill"`
	Name         string          `json:"name" doc:"任务名称"`
	NodeID       string          `json:"node_id" doc:"任务节点ID"`
	NodeName     string          `json:"node_name" doc:"任务节点名称"`
	NotFinished  bool            `json:"not_finished" doc:"本轮结束后任务是否完成"`
	IsOccupied   bool            `json:"is_occupied" doc:"是否独占，未完成时不允许命中其他task，可再增加一个全局配置"`
	IsUpdated    bool            `json:"is_updated" doc:"本轮任务是否有进展"`
	CurrentSlots []SlotNameValue `json:"current_slots" doc:"任务型当前轮收集到的词槽"`
	SessionSlots []SlotNameValue `json:"session_slots" doc:"任务型当前会话收集到的所有词槽"`
	Params       []Param         `json:"params" doc:"任务型参数"`
	LifeCycle    int             `json:"life_cycle" doc:"生命周期"`
	LatestAnswer string          `json:"latest_answer" doc:"任务型最新给出的答案"`
	Turn         int             `json:"turn" doc:"当前任务进行的轮次，从0开始计数（和DM全局轮次不一样，此处代表当前子任务（如slotfill，flow-engine）的轮次）"`
}

type SlotNameValue struct {
	Name  string `json:"name" doc:"词槽名称"`
	Value string `json:"value" doc:"词槽值"`
	Norm  string `json:"norm" doc:"词槽归一化值" spec:"optional"`
}

type Sentiment struct {
	Label int     `json:"label" doc:"0：负面、1：中性，2：正面"`
	Score float32 `json:"score" doc:"标签打分，0~1"`
	Typ   int     `json:"type" doc:"负面情绪的二级类目。-1：无意义，0：对服务态度不满，1：催促，2：商品质量不满，
3：无语生气，4：失望难过，5：脏话，6：中性，7：正面，8：其他表情(负面)，9：其他表情(正面)"`
}

type Sensitivity struct {
	Label        int    `json:"label" doc:"0：未命中，1：低等级，2：中等级，3：高等级"`
	SensitiveHit string `json:"sensitive_hit" doc:"敏感的情况下为敏感词，非敏感的情况为空"`
}

type Slot struct {
	Query    string    `json:"query"`
	Value    string    `json:"value"`
	Norm     NormValue `json:"norm"`
	Type     string    `json:"type"`
	Start    int       `json:"start"`
	End      int       `json:"end"`
	IsEntity bool      `json:"is_entity"`
	IsNumber bool      `json:"is_number"`
	Source   string    `json:"source"`
}

type NormValue interface {
	String() string
}

type EventAction struct {
	EventID uint64 `json:"event_id"`
	Action  Action `json:"action"`
}

type Action struct {
	Type   string `json:"type" doc:"动作类型" spec:"oneOf=TEXT|TEMPLATE|COMMAND"`
	Config string `json:"config" doc:"动作配置"`
}

func getRandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		result, _ := rand.Int(rand.Reader, big.NewInt(62))
		num := int(result.Int64())
		b[i] = letterBytes[num%len(letterBytes)]
	}
	return string(b)
}

func main() {
	client := http.Client{}
	input := Input{
		BizID:       103,
		SessionID:   getRandString(10),
		Query:       "华语影坛最出名的十位演员",
		NeedDump:    false,
		EmotionOn:   false,
		UserID:      "10001",
		Client:      "",
		DriveParams: nil,
		ExtraParams: nil,
	}
	body, _ := json.Marshal(input)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s:%s/%s", SSEServerIP, SSEServerPost, SSEServerPath), bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Connection", "keep-alive")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(fmt.Errorf("error making request: %v", err))
	}

	if resp.StatusCode != 200 {
		log.Fatalf("could not connect to stream: %s", http.StatusText(resp.StatusCode))
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	answers := make([]string, 0)
	for {
		if !scanner.Scan() {
			err = scanner.Err()
			if err != nil {
				log.Fatal(err)
			}
		}

		data := scanner.Text()
		if !strings.HasPrefix(data, StreamPrefix) {
			log.Printf("%v\n", data)
			continue
		}

		var resp QueryRspV2
		data = strings.TrimLeft(data, StreamPrefix)
		err := json.Unmarshal([]byte(data), &resp)
		if err != nil {
			log.Printf("Unmarshal failed, %v\n", err)
			log.Printf("%v\n", data)
		}
		log.Printf("resp:%+v\n", resp)
		// EOF 本次接收完成
		if resp.Msg == StreamEOF {
			break
		}
		// 调用出错，返回
		if resp.Msg != StreamSuccess {
			log.Printf("request err, %v", resp.Msg)
			break
		}

		if len(resp.AnswerList) > 0 {
			answers = append(answers, resp.AnswerList[0].Text)
		}
	}
	log.Printf("answers:%s\n", answers)
}

package traqmessage

import (
	"context"
	"fmt"
	"h23s_15/model"
	"strings"
	"sync"
	"time"

	"github.com/traPtitech/go-traq"
	"golang.org/x/exp/slog"
)

type MessagePoller struct {
	processor *messageProcessor
}

func NewMessagePoller() *MessagePoller {
	return &MessagePoller{
		processor: &messageProcessor{
			queue: make(chan *[]traq.Message),
		},
	}
}

// go routineの中で呼ぶこと
func (m *MessagePoller) Run() {
	go m.processor.run()

	pollingInterval := time.Minute * 3

	lastCheckpoint := time.Now()
	var checkpointMutex sync.Mutex

	ticker := time.Tick(pollingInterval)
	for range ticker {
		slog.Info("Start polling")
		func(){
			checkpointMutex.Lock()
			defer func(){
				lastCheckpoint = now
				checkpointMutex.Unlock()
			}()
	
			now := time.Now()
			collecter, err := newMessageCollecter()
			if err != nil {
				slog.Info("Skip collectMessage")
				return
			}
			var collectedMessageCount int
			for {
				messages, more, err := collecter.collectMessages(lastCheckpoint, now)
				if err != nil {
					slog.Error(fmt.Sprintf("Failled to polling messages: %v", err))
					break
				}
	
				tmpMessageCount := len(*messages)
	
				slog.Info(fmt.Sprintf("Collect %d messages", tmpMessageCount))
				collectedMessageCount += tmpMessageCount
	
				// 取得したメッセージを使っての処理の呼び出し
				m.processor.enqueue(messages)
	
				if !more {
					break
				}
			}
	
			slog.Info(fmt.Sprintf("%d messages collected totally", collectedMessageCount))
		}()
	}
}

// 通知メッセージの検索と通知処理のjobを処理する
type messageProcessor struct {
	queue chan *[]traq.Message
}

// go routineの中で呼ぶ
func (m *messageProcessor) run() {
	for {
		select {
		case messages := <-m.queue:
			m.process(*messages)
		}
	}
}

func (m *messageProcessor) enqueue(messages *[]traq.Message) {
	m.queue <- messages
}

func (m *messageProcessor) process(messages []traq.Message) {
	messageList, err := ConvertMessageHits(messages)
	if err != nil {
		slog.Error(fmt.Sprintf("Failled to convert messages: %v", err))
		return
	}
	notifyInfoList, err := model.FindMatchingWords(messageList)
	if err != nil {
		slog.Error(fmt.Sprintf("Failled to process messages: %v", err))
		return
	}

	slog.Info(fmt.Sprintf("Sending %d DMs...", len(notifyInfoList)))

	for _, notifyInfo := range notifyInfoList {
		err := sendMessage(notifyInfo.NotifyTargetTraqUuid, genNotifyMessageContent(notifyInfo.MessageId, notifyInfo.Words...))
		if err != nil {
			slog.Error(fmt.Sprintf("Failled to send message: %v", err))
			continue
		}
	}

	slog.Info("End of send DMs")
}

func genNotifyMessageContent(citeMessageId string, words ...string) string {
	list := make([]string, 0)
	for _, word := range words {
		item := fmt.Sprintf("「%s」", word)
		list = append(list, item)
	}

	return fmt.Sprintf("%s\n https://q.trap.jp/messages/%s", strings.Join(list, ""), citeMessageId)
}

func sendMessage(notifyTargetTraqUUID string, messageContent string) error {
	if model.ACCESS_TOKEN == "" {
		slog.Info("Skip sendMessage")
		return nil
	}

	client := traq.NewAPIClient(traq.NewConfiguration())
	auth := context.WithValue(context.Background(), traq.ContextAccessToken, model.ACCESS_TOKEN)
	_, _, err := client.UserApi.PostDirectMessage(auth, notifyTargetTraqUUID).PostMessageRequest(traq.PostMessageRequest{
		Content: messageContent,
	}).Execute()
	if err != nil {
		slog.Info("Error sending message: %v", err)
		return err
	}
	return nil
}

// メッセージの取得をする
type messageCollecter struct {
	client *traq.APIClient
	auth context.Context
	offset int32
}
func newMessageCollecter() (*messageCollecter, error) {
	if model.ACCESS_TOKEN == "" {
		return nil, fmt.Errorf("Error: no ACCESS_TOKEN")
	}

	return &messageCollecter{
		client: traq.NewAPIClient(traq.NewConfiguration()),
		auth: context.WithValue(context.Background(), traq.ContextAccessToken, model.ACCESS_TOKEN),
		offset: 0,
	}, nil
}

func (c *messageCollecter) collectMessages(from time.Time, to time.Time) (*[]traq.Message, bool, error) {
	// 1度での取得上限は100まで　それ以上はoffsetを使うこと
	// https://github.com/traPtitech/traQ/blob/47ed2cf94b2209c8444533326dee2a588936d5e0/service/search/engine.go#L51
	limit := 99
	result, _, err := c.client.MessageApi.SearchMessages(c.auth).After(from).Before(to).Limit(limit+1).Offset(c.offset).Execute()
	if err != nil {
		return nil, false, err
	}
	
	messages := result.Hits
	more := limit < len(messages)
	if more {
		messages = messages[:limit]
	}
	c.offset += len(messages)
	return &messages, more, nil
}

func ConvertMessageHits(messages []traq.Message) (model.MessageList, error) {
	messageList := model.MessageList{}
	for _, message := range messages {
		messageList = append(messageList, model.MessageItem{
			Id:       message.Id,
			TraqUuid: message.UserId,
			Content:  message.Content,
		})
	}
	return messageList, nil
}

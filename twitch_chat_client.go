package main

import "github.com/gempir/go-twitch-irc/v4"

const commandSignifier = "!"

type TwitchChatClient struct {
	chatConnection *twitch.Client
	channelName    string
}

func NewTwitchChatClient(userName string, oauthToken string, channelName string) *TwitchChatClient {
	twitchClient := twitch.NewClient(userName, oauthToken)

	return &TwitchChatClient{
		chatConnection: twitchClient,
		channelName:    channelName,
	}
}

func (client *TwitchChatClient) StartListening(incomingMessagesChannel chan<- ChatCommand) {
	parseIncomingMessageCallback := client.parseIncomingMessage(incomingMessagesChannel)

	client.chatConnection.OnPrivateMessage(func(message twitch.PrivateMessage) {
		go parseIncomingMessageCallback(message)
	})
}

func (client *TwitchChatClient) parseIncomingMessage(incomingMessagesChannel chan<- ChatCommand) func(message twitch.PrivateMessage) {
	return func(message twitch.PrivateMessage) {
		if command, arguments, err := ParseCommand(message.Message, commandSignifier); err == nil {
			incomingMessagesChannel <- ChatCommand{
				message.User.DisplayName,
				isModerator(message.User.Badges),
				command,
				arguments}
		}
	}
}

func (client *TwitchChatClient) StartSaying(outgoingMessagesChannel <-chan string) {
	for message := range outgoingMessagesChannel {
		client.chatConnection.Say(client.channelName, message)
	}
}

func (client *TwitchChatClient) JoinChannel() {
	client.chatConnection.Join(client.channelName)
	if err := client.chatConnection.Connect(); err != nil {
		panic(err)
	}
}

func isModerator(badges map[string]int) bool {
	return badges["broadcaster"] > 0 || badges["moderator"] > 0
}

package main

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

func ExecuteCommands(db *gorm.DB, chatCommandChannel <-chan ChatCommand, outgoingMessageChannel chan<- string) { //TODO: Replace all instances of db with a database client interface
	for chatCommand := range chatCommandChannel {
		go executeCommand(db, chatCommand, outgoingMessageChannel)
	}
}

func executeCommand(db *gorm.DB, chatCommand ChatCommand, outgoingMessageChannel chan<- string) {
	command := getCommandFromName(db, chatCommand.CommandName)
	if command.Equals(Command{}) {
		return
	}

	if command.IsModeratorOnly && !chatCommand.IsModerator {
		return
	}

	var err error

	switch command.Name {
	case IncrementCountCommandType: //TODO: add a 10 second cooldown to prevent spamming
		err = executeIncrementCountCommand(db, command.Counter)
	case IncrementCountByUserCommandType:
		err = executeIncrementCountByUserCommand(db, command.Counter, chatCommand.UserName)
	case SetCountCommandType:
		err = executeSetCountCommand(db, command.Counter, chatCommand.Arguments)
	case AddTextCommandType:
		err = executeAddTextCommand(db, chatCommand.Arguments)
	case RemoveTextCommandType:
		err = executeRemoveTextCommand(db, chatCommand.Arguments)
	case AddQuoteCommandType:
		err = executeAddQuoteCommand(db, chatCommand.Arguments)
	}

	if err != nil {
		sendFailureMessage(err, outgoingMessageChannel)
		return
	}

	sendCommandText(db, command, chatCommand, outgoingMessageChannel)
}

func getCommandFromName(db *gorm.DB, commandName string) Command {
	var command Command
	if err := db.Preload("CommandType").Preload("CommandTexts").Preload("Counter").First(&command, "name = ?", commandName).Error; err != nil {
		return Command{}
	}
	return command
}

func sendCommandText(db *gorm.DB, command Command, chatCommand ChatCommand, outgoingMessageChannel chan<- string) {
	templateVariables := getCommandTextVariables(command.CommandTexts)

	templateVariableValues := getCommandTextVariableValues(db, templateVariables, chatCommand, command)

	builtCommandTexts := getBuiltCommandTexts(command.CommandTexts, templateVariableValues)

	for _, builtCommandText := range builtCommandTexts {
		outgoingMessageChannel <- builtCommandText
	}
}

func sendFailureMessage(err error, outgoingMessageChannel chan<- string) {
	outgoingMessageChannel <- err.Error()
}

func executeIncrementCountCommand(db *gorm.DB, counter Counter) error {
	if err := db.Model(&counter).Update("count", gorm.Expr("count + ?", 1)).Error; err != nil {
		return err
	}

	return nil
}

func executeIncrementCountByUserCommand(db *gorm.DB, counter Counter, userName string) error {
	var counterByUser CounterByUser

	if err := db.FirstOrCreate(&counterByUser, CounterByUser{UserName: userName, CounterID: counter.ID}).Error; err != nil {
		return err
	}
	if err := db.Model(&counter).Update("count", gorm.Expr("count + ?", 1)).Error; err != nil {
		return err
	}
	if err := db.Model(&counterByUser).Update("count", gorm.Expr("count + ?", 1)).Error; err != nil {
		return err
	}

	return nil
}

func executeSetCountCommand(db *gorm.DB, counter Counter, commandArguments []string) error {
	if len(commandArguments) == 0 {
		return errors.New("invalid arguments")
	}

	newCount := commandArguments[0]

	if err := db.Model(&counter).Update("count", newCount).Error; err != nil {
		return err
	}

	return nil
}

func executeAddTextCommand(db *gorm.DB, commandArguments []string) error {
	if len(commandArguments) < 2 || len(strings.TrimSpace(commandArguments[0])) == 0 {
		return errors.New("invalid arguments")
	}

	commandName := commandArguments[0]

	commandText := strings.Join(commandArguments[1:], " ")

	newCommand := Command{Name: commandName,
		CommandType: CommandType{Name: UserEnteredTextCommandType},
		CommandTexts: []CommandText{
			{Text: commandText},
		},
	}

	if err := db.Create(&newCommand).Error; err != nil {
		return errors.New("command already exists")
	}

	return nil
}

func executeRemoveTextCommand(db *gorm.DB, commandArguments []string) error {
	if len(commandArguments) < 1 || len(strings.TrimSpace(commandArguments[0])) == 0 {
		return errors.New("invalid arguments")
	}

	commandName := commandArguments[0]

	command := getCommandFromName(db, commandName)

	if command.ID == 0 {
		return errors.New("command not found")
	}

	if err := db.Delete(&CommandText{}, "command_id = ?", command.ID).Error; err != nil {
		return err
	}

	if err := db.Delete(&Command{}, command).Error; err != nil {
		return err
	}

	return nil
}

func executeAddQuoteCommand(db *gorm.DB, commandArguments []string) error { //TODO: Check for quotation marks to check for name
	if len(commandArguments) < 2 || len(strings.TrimSpace(commandArguments[0])) == 0 {
		return errors.New("invalid arguments")
	}

	quoteName := commandArguments[0]

	quoteText := strings.Join(commandArguments[1:], " ")

	newQuote := Quote{Name: quoteName,
		Text: quoteText,
	}

	if err := db.Create(&newQuote).Error; err != nil {
		return errors.New("quote already exists")
	}

	return nil
}

package telegram

import (
    "context"
    "errors"
    "fmt"
    "log"
    "strconv"
    "strings"

    "read-adviser-bot/lib/e"
    "read-adviser-bot/storage"
)

const (
    StartCmd  = "/start"
    RndCmd    = "/rnd"
    HelpCmd   = "/help"
    TasksCmd  = "/tasks"
    DeleteCmd = "/delete"
)

func (p *Processor) doCmd(ctx context.Context, text string, chatID int, username string) error {
    text = strings.TrimSpace(text)

    log.Printf("got new command '%s' from '%s'", text, username)

    if isAddCmd(text) {
        return p.savePage(chatID, text, username)
    }

    switch {
    case text == RndCmd:
        return p.sendRandom(chatID, username)

    case text == HelpCmd:
        return p.sendHelp(chatID)

    case text == StartCmd:
        return p.sendHello(chatID)

    case text == TasksCmd:
        return p.sendAllTasks(ctx, chatID, username)

    case strings.HasPrefix(text, DeleteCmd):
        return p.deleteTask(ctx, chatID, username, text)

    default:
        return p.tg.SendMessage(chatID, msgUnknownCommand)
    }
}

func (p *Processor) savePage(chatID int, pageURL string, username string) (err error) {
    defer func() { err = e.WrapIfErr("can't do command: save page", err) }()

    page := &storage.Page{
        URL:      pageURL, 
        UserName: username,
    }

    isExists, err := p.storage.IsExists(context.Background(), page)
    if err != nil {
        return err
    }
    if isExists {
        return p.tg.SendMessage(chatID, msgAlreadyExists)
    }

    if err := p.storage.Save(context.Background(), page); err != nil {
        return err
    }

    if err := p.tg.SendMessage(chatID, msgSaved); err != nil {
        return err
    }

    return nil
}

func (p *Processor) sendRandom(chatID int, username string) (err error) {
    defer func() { err = e.WrapIfErr("can't do command: can't send random", err) }()

    page, err := p.storage.PickRandom(context.Background(), username)
    if err != nil && !errors.Is(err, storage.ErrNoSavedPages) {
        return err
    }
    if errors.Is(err, storage.ErrNoSavedPages) {
        return p.tg.SendMessage(chatID, msgNoSavedPages)
    }

    if err := p.tg.SendMessage(chatID, page.URL); err != nil {
        return err
    }

    return p.storage.Remove(context.Background(), page)
}

func (p *Processor) sendHelp(chatID int) error {
    return p.tg.SendMessage(chatID, msgHelp)
}

func (p *Processor) sendHello(chatID int) error {
    return p.tg.SendMessage(chatID, msgHello)
}

func isAddCmd(text string) bool {
    text = strings.TrimSpace(text)
    if text == "" {
        return false
    }
    return !strings.HasPrefix(text, "/")
}

func (p *Processor) deleteTask(ctx context.Context, chatID int, username string, text string) error {
    parts := strings.Split(text, " ")
    if len(parts) < 2 {
        return p.tg.SendMessage(chatID, "Usage: /delete <number>")
    }

    index, err := strconv.Atoi(parts[1])
    if err != nil || index < 1 {
        return p.tg.SendMessage(chatID, "Wrong number")
    }

    tasks, err := p.storage.GetAll(ctx, username)
    if err != nil {
        return p.tg.SendMessage(chatID, "You have no tasks")
    }

    if index > len(tasks) {
        return p.tg.SendMessage(chatID, "No task with this number")
    }

    taskToDelete := tasks[index-1]

    if err := p.storage.Remove(ctx, &storage.Page{
        URL:      taskToDelete.URL,
        UserName: username,
    }); err != nil {
        return p.tg.SendMessage(chatID, "Can't delete task")
    }

    return p.tg.SendMessage(chatID, fmt.Sprintf("🗑 Deleted: %s", taskToDelete.URL))
}

func (p *Processor) sendAllTasks(ctx context.Context, chatID int, username string) error {
    tasks, err := p.storage.GetAll(ctx, username)
    if err != nil {
        if errors.Is(err, storage.ErrNoSavedPages) {
            return p.tg.SendMessage(chatID, msgNoSavedPages)
        }
        log.Printf("can't get tasks: %s", err)
        return e.Wrap("can't get tasks", err)
    }

    if len(tasks) == 0 {
        return p.tg.SendMessage(chatID, msgNoSavedPages)
    }

    var b strings.Builder
    b.WriteString("📝 Your tasks:\n\n")

    for i, t := range tasks {
        line := fmt.Sprintf("%d. %s\n", i+1, t.URL)
        b.WriteString(line)
    }

    return p.tg.SendMessage(chatID, b.String())
}

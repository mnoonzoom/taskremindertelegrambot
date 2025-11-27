package telegram

const msgHelp = `I can save and keep your tasks.

To save a task, just send me a message like:
- do homework
- clean the room
- buy groceries

To get a random task from your list, send /rnd.
To see ALL your tasks in one message, send /tasks.`

const msgHello = "Hi there! ✅\n\n" + msgHelp

const (
    msgUnknownCommand = "Unknown command 🤔"
    msgNoSavedPages   = "You have no saved tasks 🙊"
    msgSaved          = "Task saved! 👌"
    msgAlreadyExists  = "You already have this task in your list 🤗"
)
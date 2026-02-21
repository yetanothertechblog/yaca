package slashcmd

func init() {
	Register(Command{"/resume", "Resume a previous conversation"})
}

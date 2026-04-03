package main

type RUNTIME_OPT struct {
	argStop		bool
	applicationArgs	[]string
	userExpose	chan map[string]string
	userLang	string
	miTerminate	bool
	writtenDesktop	bool
	isDebug		bool
}
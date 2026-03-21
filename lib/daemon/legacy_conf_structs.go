package main

type portableConfigOpts struct {
	confPath		string
	networkDeny		string
	friendlyName		string
	appID			string
	stateDirectory		string
	launchTarget		string	// this one may be empty?
	busLaunchTarget		string	// also may be empty
	bindNetwork		bool
	terminateImmediately	bool
	allowClassicNotifs	bool
	useZink			bool
	qt5Compat		bool
	waylandOnly		bool
	gameMode		bool
	mprisName		string // may be empty
	bindCameras		bool
	bindPipewire		bool
	bindInputDevices	bool
	allowInhibit		bool
	allowGlobalShortcuts	bool
	allowKDEStatus		bool
	dbusWake		bool
	mountInfo		bool
}

type confTarget struct {
	str			*string
	b			*bool
}

// Defaults should be set in readConf()
var targets = map[string]confTarget{
	"appID":		{str: &confOpts.appID},
	"friendlyName":		{str: &confOpts.friendlyName},
	"stateDirectory":	{str: &confOpts.stateDirectory},
	"launchTarget":		{str: &confOpts.launchTarget},
	"busLaunchTarget":	{str: &confOpts.busLaunchTarget},
	"mprisName":		{str: &confOpts.mprisName},
	"bindNetwork":		{b: &confOpts.bindNetwork},
	"terminateImmediately":	{b: &confOpts.terminateImmediately},
	"networkDeny":		{str: &confOpts.networkDeny},
	"allowClassicNotifs":	{b: &confOpts.allowClassicNotifs},
	"useZink":		{b: &confOpts.useZink},
	"qt5Compat":		{b: &confOpts.qt5Compat},
	"waylandOnly":		{b: &confOpts.waylandOnly},
	"gameMode":		{b: &confOpts.gameMode},
	"bindCameras":		{b: &confOpts.bindCameras},
	"bindPipewire":		{b: &confOpts.bindPipewire},
	"bindInputDevices":	{b: &confOpts.bindInputDevices},
	"allowInhibit":		{b: &confOpts.allowInhibit},
	"allowGlobalShortcuts":	{b: &confOpts.allowGlobalShortcuts},
	"allowKDEStatus":	{b: &confOpts.allowKDEStatus},
	"dbusWake":		{b: &confOpts.dbusWake},
	"mountInfo":		{b: &confOpts.mountInfo},
}

var confInfo = map[string]string{
	"appID":		"string",
	"friendlyName":		"string",
	"stateDirectory":	"string",
	"launchTarget":		"string",
	"busLaunchTarget":	"string",
	"bindNetwork":		"bool",
	"terminateImmediately":	"bool",
	"networkDeny":		"string",
	"allowClassicNotifs":	"bool",
	"useZink":		"bool",
	"qt5Compat":		"bool",
	"waylandOnly":		"bool",
	"gameMode":		"bool",
	"mprisName":		"string",
	"bindCameras":		"bool",
	"bindPipewire":		"bool",
	"bindInputDevices":	"bool",
	"allowInhibit":		"bool",
	"allowGlobalShortcuts":	"bool",
	"allowKDEStatus":	"bool",
	"dbusWake":		"bool",
	"mountInfo":		"bool",
}
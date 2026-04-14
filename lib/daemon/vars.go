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

const (
	version		float32	=	15.99
)

type RUNTIME_PARAMS struct {
	instanceID		string
	bwCmd			[]string
}

type XDG_DIRS struct {
	runtimeDir		string
	confDir			string
	cacheDir		string
	dataDir			string
	home			string
}

type PassFiles struct {
	// FileMap is a map that contains [host path string](docid string)
	FileMap		map[string]string
}

var (
	internalLoggingLevel	int
	runtimeInfo		RUNTIME_PARAMS
	xdgDir			XDG_DIRS
	runtimeOpt		RUNTIME_OPT
	envsChan		= make(chan string, 512)
	envsFlushReady		= make(chan int8, 1)
	// When true and present, aborts start before multiinstance detection
	abortChan		= make(chan bool, 10)
	gpuChan 		= make(chan []string, 1)
	busArgChan		= make(chan []string, 1)
	socketStop		= make(chan int8, 10)
	stopAppChan		= make(chan int8, 512)
	stopAppDone		= make(chan int8)
	nvKernelModulePath 	= []string{
					"/sys/module/nvidia",
					"/sys/module/nvidia_drm",
					"/sys/module/nvidia_modeset",
					"/sys/module/nvidia_uvm",
					"/sys/module/nvidia_wmi_ec_backlight",
				}
	filesInfo		= make(chan PassFiles, 1)
)
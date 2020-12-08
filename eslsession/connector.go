package eslsession

import (
	"fmt"

	fs "github.com/babakyakhchali/go-esl-wrapper/fs"
	l "github.com/babakyakhchali/go-esl-wrapper/logger"
	"github.com/google/uuid"
)

//FsConnector acts as a channel between fs and session
type FsConnector struct {
	uuid string
	//used to send api and execute to freeswitch
	cmds chan map[string]string
	/*used to recieve events by session dispatcher.
	this will receive both exec result events and other channel events by dispatcher*/
	events chan fs.IEvent
	//receives errors from fs connection
	errors chan error

	/*used by dispatcher to notify the exec() when execution completes*/
	execEvent chan fs.IEvent
	execError chan error

	jobEvent       chan fs.IEvent
	jobError       chan error
	currentAppUUID string
	closed         bool
	logger         *l.NsLogger
	EventHandlers  map[string]fs.EventHandlerFunc

	currentJobUUID string
}

func (fs *FsConnector) close() {
	fs.closed = true
	close(fs.cmds)
}

//sits between event channel and session and receives all events and replies for the session
func (fs *FsConnector) dispatch() {
	for {
		select {
		case event := <-fs.events:
			ename := event.GetHeader("Event-Name")
			fs.logger.Debug("dispatch(): got event %s:%s", ename, fs.uuid)
			euuid := event.GetHeader("Application-UUID")
			if ename == "CHANNEL_EXECUTE_COMPLETE" && euuid == fs.currentAppUUID {
				select { //this must be nonblocking
				case fs.execEvent <- event:
				default:
				}
			}
			if ename == "CHANNEL_DESTROY" {
				fs.close()
				select { //this must be nonblocking
				case fs.execError <- fmt.Errorf(EChannelClosed):
				default:
				}
				fs.logger.Debug("dispatch(): ended by CHANNEL_DESTROY")
				return
			}
			if h, e := fs.EventHandlers[ename]; e {
				go h(event)
			}
		case err := <-fs.errors: //inform blocked execs and bgapis
			select {
			case fs.execError <- err:
			default:
			}
			select {
			case fs.jobError <- err:
			default:
			}
			fs.close()
			fs.logger.Debug("dispatch(): ended by error:", err)
			return

		}
	}

}

//Application-UUID Event-UUID
//
//this method handles complex logic because of the event based nature of the module
//channel may be in 3 states when this method is called on session:
//
// * already hangged up
// * in the middle of hangup
// * up and running
func (fs *FsConnector) exec(app string, args string) (fs.IEvent, error) {
	if fs.closed {
		return nil, fmt.Errorf(EChannelClosed)
	}
	headers := make(map[string]string)
	headers["call-command"] = "execute"
	headers["execute-app-name"] = app
	headers["execute-app-arg"] = args
	headers["Event-UUID"] = uuid.New().String()
	fs.currentAppUUID = headers["Event-UUID"]

	defer func() {
		fs.currentAppUUID = ""
	}()

	fs.cmds <- headers

	select {
	case event := <-fs.execEvent:
		return event, nil
	case err := <-fs.execError:
		fs.logger.Debug("exec(%s,%s)(%s) error: %s", app, args, fs.currentAppUUID, err)
		return nil, err
	}
}

func (fs *FsConnector) bgapi(cmd string) (fs.IEvent, error) {
	if fs.closed {
		return nil, fmt.Errorf(EChannelClosed)
	}
	headers := make(map[string]string)
	headers["bgapi"] = cmd
	headers["Job-UUID"] = uuid.New().String()
	fs.currentJobUUID = headers["Job-UUID"]

	defer func() {
		fs.currentJobUUID = ""
	}()

	fs.cmds <- headers

	select {
	case event := <-fs.jobEvent:
		fs.logger.Debug("bgapi(%s) => %s", cmd, event.GetBody())
		return event, nil
	case err := <-fs.jobError:
		fs.logger.Debug("bgapi(%s) error: %s", cmd, err)
		return nil, err
	}
}

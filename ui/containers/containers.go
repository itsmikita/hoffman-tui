package containers

import (
	"fmt"
	"strings"
	"sync"

	"github.com/containers/podman/v3/pkg/domain/entities"
	"github.com/containers/podman-tui/ui/containers/cntdialogs"
	"github.com/containers/podman-tui/ui/dialogs"
	"github.com/containers/podman-tui/ui/utils"
	"github.com/rivo/tview"
)

// Containers implements the containers page primitive
type Containers struct {
	*tview.Box
	title          string
	headers        []string
	table          *tview.Table
	errorDialog    *dialogs.ErrorDialog
	cmdDialog      *dialogs.CommandDialog
	cmdInputDialog *dialogs.SimpleInputDialog
	confirmDialog  *dialogs.ConfirmDialog
	messageDialog  *dialogs.MessageDialog
	progressDialog *dialogs.ProgressDialog
	topDialog      *dialogs.TopDialog
	createDialog   *cntdialogs.ContainerCreateDialog
	containersList containerListReport
	selectedID     string
	selectedName   string
	confirmData    string
}

type containerListReport struct {
	mu     sync.Mutex
	report []entities.ListContainer
}

// NewContainers returns containers page view
func NewContainers() *Containers {
	containers := &Containers{
		Box:            tview.NewBox(),
		title:          "containers",
		headers:        []string{"container id", "image", "pod", "created", "status", "names", "ports"},
		errorDialog:    dialogs.NewErrorDialog(),
		cmdInputDialog: dialogs.NewSimpleInputDialog(""),
		messageDialog:  dialogs.NewMessageDialog(""),
		progressDialog: dialogs.NewProgressDialog(),
		confirmDialog:  dialogs.NewConfirmDialog(),
		topDialog:      dialogs.NewTopDialog(),
		createDialog:   cntdialogs.NewContainerCreateDialog(),
	}
	containers.topDialog.SetTitle("podman container top")

	containers.cmdDialog = dialogs.NewCommandDialog([][]string{
		{"create", "create a new container but do not start"},
		{"diff", "inspect changes to the selected container's file systems"},
		{"inspect", "display the configuration of a container"},
		{"kill", "kill the selected running container with a SIGKILL signal"},
		//{"init", "initialize the selected container"},
		{"logs", "fetch the logs of the selected container"},
		{"pause", "pause all the processes in the selected container"},
		{"port", "list port mappings for the selected container"},
		{"prune", "remove all non running containers"},
		{"rename", "rename the selected container"},
		{"rm", "remove the selected container"},
		//{"run", "run a command in a new container"},
		{"start", "start the selected containers"},
		//{"stats", "display a live stream of container resource usage statistics"},
		{"stop", "stop the selected containers"},
		{"top", "display the running processes of the selected container"},
		{"unpause", "unpause the selected container that was paused before"},
	})

	fgColor := utils.Styles.PageTable.FgColor
	bgColor := utils.Styles.PageTable.BgColor
	containers.table = tview.NewTable()
	containers.table.SetTitle(fmt.Sprintf("[::b]%s[0]", strings.ToUpper(containers.title)))
	containers.table.SetBorderColor(bgColor)
	containers.table.SetTitleColor(fgColor)
	containers.table.SetBorder(true)
	fgColor = utils.Styles.PageTable.HeaderRow.FgColor
	bgColor = utils.Styles.PageTable.HeaderRow.BgColor

	for i := 0; i < len(containers.headers); i++ {
		containers.table.SetCell(0, i,
			tview.NewTableCell(fmt.Sprintf("[black::b]%s", strings.ToUpper(containers.headers[i]))).
				SetExpansion(1).
				SetBackgroundColor(bgColor).
				SetTextColor(fgColor).
				SetAlign(tview.AlignLeft).
				SetSelectable(false))
	}

	containers.table.SetFixed(1, 1)
	containers.table.SetSelectable(true, false)

	// set command dialog functions
	containers.cmdDialog.SetSelectedFunc(func() {
		containers.cmdDialog.Hide()
		containers.runCommand(containers.cmdDialog.GetSelectedItem())
	})
	containers.cmdDialog.SetCancelFunc(func() {
		containers.cmdDialog.Hide()
	})
	// set input cmd dialog functions
	containers.cmdInputDialog.SetCancelFunc(func() {
		containers.cmdInputDialog.Hide()
	})

	containers.cmdInputDialog.SetSelectedFunc(func() {
		containers.cmdInputDialog.Hide()
	})
	// set message dialog functions
	containers.messageDialog.SetSelectedFunc(func() {
		containers.messageDialog.Hide()
	})
	containers.messageDialog.SetCancelFunc(func() {
		containers.messageDialog.Hide()
	})

	// set container top dialog functions
	containers.topDialog.SetDoneFunc(func() {
		containers.topDialog.Hide()
	})

	// set confirm dialogs functions
	containers.confirmDialog.SetSelectedFunc(func() {
		containers.confirmDialog.Hide()
		switch containers.confirmData {
		case "prune":
			containers.prune()
		case "rm":
			containers.remove()
		}
	})
	containers.confirmDialog.SetCancelFunc(func() {
		containers.confirmDialog.Hide()
	})
	// set create dialog functions
	containers.createDialog.SetCancelFunc(func() {
		containers.createDialog.Hide()
	})
	containers.createDialog.SetCreateFunc(func() {
		containers.createDialog.Hide()
		containers.create()
	})
	return containers
}

// GetTitle returns primitive title
func (cnt *Containers) GetTitle() string {
	return cnt.title
}

// HasFocus returns whether or not this primitive has focus
func (cnt *Containers) HasFocus() bool {
	if cnt.table.HasFocus() || cnt.errorDialog.HasFocus() {
		return true
	}
	if cnt.cmdDialog.HasFocus() || cnt.progressDialog.HasFocus() {
		return true
	}
	if cnt.topDialog.HasFocus() || cnt.messageDialog.IsDisplay() {
		return true
	}
	if cnt.confirmDialog.HasFocus() || cnt.cmdInputDialog.IsDisplay() {
		return true
	}
	if cnt.createDialog.HasFocus() {
		return true
	}
	return cnt.Box.HasFocus()
}

// Focus is called when this primitive receives focus
func (cnt *Containers) Focus(delegate func(p tview.Primitive)) {
	// error dialog
	if cnt.errorDialog.IsDisplay() {
		delegate(cnt.errorDialog)
		return
	}
	// command dialog
	if cnt.cmdDialog.IsDisplay() {
		delegate(cnt.cmdDialog)
		return
	}
	// command input dialog
	if cnt.cmdInputDialog.IsDisplay() {
		delegate(cnt.cmdInputDialog)
		return
	}
	// message dialog
	if cnt.messageDialog.IsDisplay() {
		delegate(cnt.messageDialog)
		return
	}
	// container top dialog
	if cnt.topDialog.IsDisplay() {
		delegate(cnt.topDialog)
		return
	}
	// confirm dialog
	if cnt.confirmDialog.IsDisplay() {
		delegate(cnt.confirmDialog)
		return
	}
	// create dialog
	if cnt.createDialog.IsDisplay() {
		delegate(cnt.createDialog)
		return
	}
	delegate(cnt.table)
}

func (cnt *Containers) getSelectedItem() (string, string) {
	var cntID string
	var cntName string
	if cnt.table.GetRowCount() <= 1 {
		return cntID, cntName
	}
	row, _ := cnt.table.GetSelection()
	cntID = cnt.table.GetCell(row, 0).Text
	cntName = cnt.table.GetCell(row, 5).Text
	return cntID, cntName
}
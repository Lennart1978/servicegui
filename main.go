package main

import (
	"fmt"
	"log"
	"os/exec"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/coreos/go-systemd/v22/dbus"
)

var data = [][]string{{"UNIT:", "LOAD:", "ACTIVE:", "SUB:", "DESCRIPTION:"}}

type service struct {
	Name        string
	LoadState   string
	ActiveState string
	SubState    string
	Description string
}

var selectedService string
var show string

func updateTable(list *widget.Table) {
	var services []service
	srvs, err := listRunningServices()
	if err != nil {
		log.Fatal(err)
	}

	data = [][]string{{"UNIT:", "LOAD:", "ACTIVE:", "SUB:", "DESCRIPTION:"}} // Daten zurücksetzen

	for i := 0; i < len(srvs); i += 5 {
		switch show {
		case "all":
			services = append(services, service{Name: srvs[i], LoadState: srvs[i+1], ActiveState: srvs[i+2], SubState: srvs[i+3], Description: srvs[i+4]})

		case "active":
			if srvs[i+2] == "active" {
				services = append(services, service{Name: srvs[i], LoadState: srvs[i+1], ActiveState: srvs[i+2], SubState: srvs[i+3], Description: srvs[i+4]})
			}

		case "inactive":
			if srvs[i+2] == "inactive" {
				services = append(services, service{Name: srvs[i], LoadState: srvs[i+1], ActiveState: srvs[i+2], SubState: srvs[i+3], Description: srvs[i+4]})
			}
		}
	}

	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	for _, s := range services {
		data = append(data, []string{s.Name, s.LoadState, s.ActiveState, s.SubState, s.Description})
	}

	list.Refresh() // Aktualisiert die Tabelle
}

func stopService(serviceName string) error {
	conn, err := dbus.NewSystemConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	// Der zweite Parameter ist die Methode, wie der Dienst gestoppt werden soll.
	// "replace" ist eine gängige Option, die eine laufende Einheit durch eine andere ersetzt oder stoppt, falls keine Ersatzeinheit angegeben wird.
	reschan := make(chan string)
	_, err = conn.StopUnit(serviceName, "replace", reschan)
	if err != nil {
		return err
	}

	// Warten auf das Ergebnis des Stop-Befehls
	result := <-reschan
	if result != "done" {
		return fmt.Errorf("fehler beim Stoppen des Dienstes: %s", result)
	}
	fmt.Printf("Dienst %s erfolgreich gestoppt !", serviceName)
	return nil
}

func restartService(serviceName string) error {
	// Logik zum Neustarten eines Dienstes
	conn, err := dbus.NewSystemConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	// Der zweite Parameter ist die Methode, wie der Dienst gestoppt werden soll.
	// "replace" ist eine gängige Option, die eine laufende Einheit durch eine andere ersetzt oder stoppt, falls keine Ersatzeinheit angegeben wird.
	reschan := make(chan string)
	_, err = conn.RestartUnit(serviceName, "replace", reschan)
	if err != nil {
		return err
	}

	// Warten auf das Ergebnis des Stop-Befehls
	result := <-reschan
	if result != "done" {
		return fmt.Errorf("fehler beim Restarten des Dienstes: %s", result)
	}
	fmt.Printf("Dienst %s erfolgreich neu gestartet !", serviceName)
	return nil
}

func removeService(serviceName string) error {
	// Pfad zur Systemd-Unit-Datei (ersetzen Sie dies durch den tatsächlichen Pfad, falls nötig)
	unitFilePath := fmt.Sprintf("/etc/systemd/system/%s", serviceName)

	// Entfernen der Unit-Datei
	removeCmd := exec.Command("sudo", "rm", unitFilePath)
	err := removeCmd.Run()
	if err != nil {
		return fmt.Errorf("fehler beim Entfernen der Unit-Datei: %s", err)
	}

	// Neuladen von systemd, um die Änderungen zu übernehmen
	reloadCmd := exec.Command("sudo", "systemctl", "daemon-reload")
	err = reloadCmd.Run()
	if err != nil {
		return fmt.Errorf("fehler beim Neuladen von systemd: %s", err)
	}

	fmt.Printf("Dienst %s erfolgreich entfernt und systemd neugeladen\n", serviceName)
	return nil
}

func listRunningServices() ([]string, error) {
	conn, err := dbus.NewSystemConnection()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	units, err := conn.ListUnits()
	if err != nil {
		return nil, err
	}

	var services []string
	for _, u := range units {
		// Überprüfen, ob der Name des Dienstes mit ".service" endet
		if strings.HasSuffix(u.Name, ".service") {
			services = append(services, u.Name, u.LoadState, u.ActiveState, u.SubState, u.Description)
		}
	}

	return services, nil
}
func main() {
	var services []service

	srvs, err := listRunningServices()
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < len(srvs); i += 5 {
		services = append(services, service{Name: srvs[i], LoadState: srvs[i+1], ActiveState: srvs[i+2], SubState: srvs[i+3], Description: srvs[i+4]})
	}

	sort.Slice(services, func(i, j int) bool {
		return strings.ToLower(services[i].Name) < strings.ToLower(services[j].Name)
	})

	for _, s := range services {
		data = append(data, []string{s.Name, s.LoadState, s.ActiveState, s.SubState, s.Description})
	}

	myApp := app.NewWithID("com.lennart.servicegui")
	myWindow := myApp.NewWindow("Services")
	myWindow.SetFullScreen(true)

	list := widget.NewTable(
		func() (int, int) {
			return len(data), len(data[0])
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("wide content")
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(data[i.Row][i.Col])
		})

	list.SetColumnWidth(0, 300)
	list.SetColumnWidth(4, 300)

	// Buttons
	buttonStop := widget.NewButton("Stop", func() {
		if err := stopService(selectedService); err != nil {
			log.Println("Error stopping service:", err)
			dialog.ShowError(err, myWindow)
		} else {
			updateTable(list)
			dialog.ShowInformation("Info", fmt.Sprintf("Service %s stopped successfully", selectedService), myWindow)
		}
	})

	buttonRestart := widget.NewButton("Restart", func() {
		log.Println("Restarted")
		if err := restartService(selectedService); err != nil {
			log.Println("Error restarting service:", err)
			dialog.ShowError(err, myWindow)
		} else {
			updateTable(list)
			dialog.ShowInformation("Info", fmt.Sprintf("Service %s restarted successfully", selectedService), myWindow)
		}
	})

	buttonRemove := widget.NewButton("Remove (must be root !)", func() {
		log.Println("Removed")
		if err := removeService(selectedService); err != nil {
			log.Println("Error removing service:", err)
			dialog.ShowError(err, myWindow)
		} else {
			updateTable(list)
			dialog.ShowInformation("Info", fmt.Sprintf("Service %s removed successfully", selectedService), myWindow)
		}
	})

	buttonAbout := widget.NewButtonWithIcon("", theme.NewThemedResource(theme.HelpIcon()), func() {
		dialog.ShowInformation("About", "(c)2023 Lennart Martens\nLicense: MIT\ngithub.com/lennart1978/servicegui\nmonkeynator78@gmail.com", myWindow)

	})

	buttonExit := widget.NewButtonWithIcon("Quit", theme.NewThemedResource(theme.CancelIcon()), func() {
		myWindow.Close()
		myApp.Quit()
	})

	// Label
	labelShow := widget.NewLabel("Show: ")

	// Combo
	comboActive := widget.NewSelect([]string{"active", "inactive", "all"}, func(value string) {
		log.Println("Select set to", value)
	})
	comboActive.SetSelected("all")
	show = "all"
	comboActive.OnChanged = func(value string) {
		log.Println("Select changed to", value)
		show = value
		updateTable(list)
	}

	list.OnSelected = func(id widget.TableCellID) {
		selectedService = data[id.Row][0]
		log.Println("Selected service:", selectedService)
	}

	//container
	contTop := container.NewHBox(buttonStop, buttonRestart, labelShow, comboActive, buttonRemove, buttonAbout, buttonExit)
	cont := container.NewBorder(contTop, nil, nil, nil, list)
	myWindow.SetContent(cont)
	myWindow.ShowAndRun()
}

package main

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	listView uint = iota
	nameView
	methodView
	endpointView
	payloadView
	requestDelayView
	jsonPropertyView
	expectedValueView
	preferredStatusView
	insecureSkipVerifyView
)

type model struct {
	store        *Store
	state        uint
	textinput    textinput.Model
	spinner      spinner.Model
	pulseSpinner spinner.Model
	currService  Service
	services     []Service
	listIndex    int
	errorMsg     string
}

// type tickMsg time.Time
type dataMsg *[]Service

func NewModel(store *Store) model {
	services, err := store.GetServices()
	if err != nil {
		log.Fatalf("Unable to get services: %v", err)
	}

	s := spinner.New()
	s.Spinner = spinner.Ellipsis
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ps := spinner.New()
	ps.Spinner = spinner.Pulse

	return model{
		store:        store,
		state:        listView,
		textinput:    textinput.New(),
		spinner:      s,
		pulseSpinner: ps,
		services:     services,
		errorMsg:     "",
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.pulseSpinner.Tick,
		refreshServices(m),
	)
}

func getStatuses(services []Service) []Service {
	for i := range services {
		status := getStatus(services[i])

		// Add current status value to StatusHistory
		limit := 20
		lenght := len(services[i].StatusHistory) + 1
		if lenght >= limit {
			services[i].StatusHistory = services[i].StatusHistory[:limit-1]
		}
		services[i].StatusHistory = append([]bool{status}, services[i].StatusHistory...)

		// Create status bar info
		statusBar := "- "
		if status {
			statusBar += lipgloss.NewStyle().Background(lipgloss.Color("2")).Padding(0, 1).Render("Online")
		} else {
			statusBar += lipgloss.NewStyle().Background(lipgloss.Color("1")).Padding(0, 1).Render("Offline")
		}

		// Health bar
		for _, v := range services[i].StatusHistory {
			if v {
				statusBar += " " + lipgloss.NewStyle().Background(lipgloss.Color("2")).Render(" ")
			} else {
				statusBar += " " + lipgloss.NewStyle().Background(lipgloss.Color("1")).Render(" ")
			}
		}

		services[i].LastStatusInfo = statusBar
	}
	return services
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
		cmd  tea.Cmd
	)

	m.textinput, cmd = m.textinput.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case dataMsg:
		m.services = []Service(*msg)
		return m, tea.Tick(5*time.Second, func(_ time.Time) tea.Msg {
			m.services = getStatuses(m.services)
			return dataMsg(&m.services)
		})
	case tea.KeyMsg:
		key := msg.String()
		switch m.state {
		case listView:
			switch key {
			case "q":
				return m, tea.Quit
			case "n":
				m.textinput.SetValue("")
				m.textinput.Focus()
				m.currService = Service{}
				m.state = nameView
			case "d":
				m.store.DeleteService(m.services[m.listIndex])
				m.state = listView
				if m.listIndex >= len(m.services)-1 && m.listIndex > 0 {
					m.listIndex--
				}
				return m, refreshServices(m)
			case "up", "k":
				if m.listIndex > 0 {
					m.listIndex--
				}
			case "down", "j":
				if m.listIndex < len(m.services)-1 {
					m.listIndex++
				}
			case "enter":
				m.currService = m.services[m.listIndex]
				m.state = nameView
				m.textinput.SetValue(m.currService.Name)
				m.textinput.Focus()
				m.textinput.CursorEnd()
			case "r":
				m.services[m.listIndex].StatusHistory = m.services[m.listIndex].StatusHistory[:0]
				return m, refreshServices(m)
			}

		case nameView:
			switch key {
			case "enter":
				m.errorMsg = ""
				name := strings.TrimSpace(m.textinput.Value())
				if name == "" {
					m.errorMsg = "Service name cannot be empty"
					break
				}
				m.currService.Name = name
				m.state = methodView
				m.SetFieldValue("Method")
			case "esc":
				m.state = listView
			}

		case methodView:
			switch key {
			case "enter":
				m.errorMsg = ""
				method := strings.ToUpper(strings.TrimSpace(m.textinput.Value()))
				if method == "" || (method != "GET" && method != "POST" && method != "PUT" && method != "DELETE" && method != "PATCH" && method != "HEAD" && method != "OPTIONS") {
					m.errorMsg = "Invalid method (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS)"
					break
				}
				m.currService.Method = method
				m.state = endpointView
				m.SetFieldValue("Endpoint")
			case "esc":
				m.state = nameView
			}

		case endpointView:
			switch key {
			case "enter":
				m.errorMsg = ""
				endpoint := strings.TrimSpace(m.textinput.Value())
				if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
					m.errorMsg = "Invalid endpoint (http:// or https://)"
					break
				}
				m.currService.Endpoint = endpoint
				m.state = payloadView
				m.SetFieldValue("Payload")
			case "esc":
				m.state = methodView
			}

		case payloadView:
			switch key {
			case "enter":
				m.errorMsg = ""
				payload := strings.TrimSpace(m.textinput.Value())
				// Payload can be empty, so we allow it as-is
				m.currService.Payload = payload
				m.state = requestDelayView
				m.SetFieldValue("RequestDelay")
			case "esc":
				m.state = endpointView
				m.SetFieldValue("Endpoint")
			}

		case requestDelayView:
			switch key {
			case "enter":
				m.errorMsg = ""
				requestDelay := strings.TrimSpace(m.textinput.Value())
				// Request delay can be empty, so we allow it as-is
				if requestDelay != "" {
					if _, err := strconv.Atoi(requestDelay); err != nil {
						m.errorMsg = "Invalid request delay (integer)"
						break
					}
				}
				m.currService.RequestDelay = requestDelay
				m.state = jsonPropertyView
				m.SetFieldValue("JSONProperty")
			case "esc":
				m.state = endpointView
				m.SetFieldValue("Payload")
			}

		case jsonPropertyView:
			switch key {
			case "enter":
				m.errorMsg = ""
				jsonProperty := strings.TrimSpace(m.textinput.Value())
				m.currService.JSONProperty = jsonProperty
				if jsonProperty != "" {
					m.state = expectedValueView
					m.SetFieldValue("ExpectedValue")
				} else {
					m.state = preferredStatusView
					m.SetFieldValue("PreferredStatus")
				}
			case "esc":
				m.state = endpointView
				m.SetFieldValue("Endpoint")
			}

		case expectedValueView:
			switch key {
			case "enter":
				m.errorMsg = ""
				expectedValue := strings.TrimSpace(m.textinput.Value())
				m.currService.ExpectedValue = expectedValue
				m.state = insecureSkipVerifyView
				m.SetFieldValue("InsecureSkipVerify")
			case "esc":
				m.state = jsonPropertyView
				m.SetFieldValue("JSONProperty")
			}

		case preferredStatusView:
			switch key {
			case "enter":
				m.errorMsg = ""
				preferredStatus := strings.TrimSpace(m.textinput.Value())
				if preferredStatus == "" {
					m.errorMsg = "Preferred status cannot be empty (100-599)"
					break
				}
				statusCode, err := strconv.Atoi(preferredStatus)
				if err != nil || statusCode < 100 || statusCode > 599 {
					m.errorMsg = "Invalid preferred status (100-599)"
					break
				}
				m.currService.PreferredStatus = preferredStatus
				m.state = insecureSkipVerifyView
				m.SetFieldValue("InsecureSkipVerify")
			case "esc":
				m.state = jsonPropertyView
				m.SetFieldValue("JSONProperty")
			}

		case insecureSkipVerifyView:
			switch key {
			case "enter":
				m.errorMsg = ""
				insecureSkipVerify := strings.TrimSpace(m.textinput.Value())
				lower := strings.ToLower(insecureSkipVerify)
				if lower != "true" && lower != "false" {
					m.errorMsg = "Invalid insecure skip verify (true/false)"
					break
				}
				m.currService.InsecureSkipVerify = lower
				m.store.SaveService(m.currService)
				m.state = listView
				return m, refreshServices(m)
			case "esc":
				m.state = preferredStatusView
				m.SetFieldValue("PreferredStatus")
			}
		}
	default:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
		m.pulseSpinner, cmd = m.pulseSpinner.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}
	return m, tea.Batch(cmds...)
}

func refreshServices(m model) tea.Cmd {
	services, err := m.store.GetServices()
	if err != nil {
		log.Fatalf("Unable to get services: %v", err)
	}
	services = getStatuses(services)
	return func() tea.Msg {
		return dataMsg(&services)
	}
}

func (m *model) SetFieldValue(p string) {
	v := reflect.ValueOf(m.currService)
	field := v.FieldByName(p)
	var value string
	if field.IsValid() && field.Kind() == reflect.String {
		value = field.String()
	}
	if value != "" {
		m.textinput.SetValue(value)
	} else {
		m.textinput.SetValue("")
	}
	m.textinput.Focus()
	m.textinput.CursorEnd()
}

func getStatus(s Service) bool {
	if len(s.StatusHistory) > 0 {
		delay := 0
		requestDelay := s.RequestDelay
		if requestDelay != "" {
			conv, err := strconv.Atoi(requestDelay)
			if err == nil {
				delay = conv
			}
		}

		time.Sleep(time.Duration(time.Duration(delay).Milliseconds()))
	}

	isv := s.InsecureSkipVerify == "true"
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: isv},
	}
	client := &http.Client{Transport: tr}

	status := false
	var resp *http.Response
	var err error
	if strings.ToUpper(strings.TrimSpace(s.Method)) == "GET" {
		resp, err = client.Get(s.Endpoint)
		status = err == nil
	}
	defer resp.Body.Close()

	if s.JSONProperty != "" && resp != nil && err == nil {
		var jsonData map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&jsonData); err == nil {
			keys := strings.Split(s.JSONProperty, ".")
			var v interface{} = jsonData
			jsonPropertyExists := true

			for _, k := range keys {
				m, ok := v.(map[string]interface{})
				if !ok {
					jsonPropertyExists = false
					break
				}
				v, ok = m[k]
				if !ok {
					jsonPropertyExists = false
					break
				}
			}

			// Only set status to true if we successfully navigated through all keys
			status = jsonPropertyExists && (v == s.ExpectedValue || s.ExpectedValue == "")
		} else {
			status = false
		}
	}

	var preferredStatus int
	preferredStatus, err = strconv.Atoi(s.PreferredStatus)
	if err != nil {
		preferredStatus = 200
	}
	return status && resp.StatusCode == int(preferredStatus)
}

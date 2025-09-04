package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	appNameStyle = lipgloss.NewStyle().Padding(1, 0, 2)
	appSubStyle  = lipgloss.NewStyle().Padding(1, 2).Foreground(lipgloss.Color("#0089F9"))

	faint = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Faint(true)

	listEnumeratorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#0089F9")).MarginRight(1)
	errorMessageStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Faint(true)
	helperStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Faint(true)
)

func (m model) View() string {
	s := appNameStyle.Render("Welcome to goardian 🛡")
	s += appSubStyle.Render("HTTP service health checker") + "\n\n"

	if m.state == nameView {
		s += "Service name: \n\n"
		s += m.textinput.View() + "\n\n"
		s += helperStyle.Render("Enter service name") + "\n\n"
	}

	if m.state == methodView {
		s += "Method: \n\n"
		s += m.textinput.View() + "\n\n"
		s += helperStyle.Render("Enter HTTP method (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS)") + "\n\n"
	}

	if m.state == endpointView {
		s += "Endpoint: \n\n"
		s += m.textinput.View() + "\n\n"
		s += helperStyle.Render("Enter HTTP endpoint (http:// or https://)") + "\n\n"
	}

	if m.state == payloadView {
		s += "Payload: \n\n"
		s += m.textinput.View() + "\n\n"
		s += helperStyle.Render("Enter HTTP payload (JSON)") + "\n\n"
	}

	if m.state == requestDelayView {
		s += "Request delay: \n\n"
		s += m.textinput.View() + "\n\n"
		s += helperStyle.Render("Enter HTTP request delay (milliseconds)") + "\n\n"
	}

	if m.state == jsonPropertyView {
		s += "JSON Property: \n\n"
		s += m.textinput.View() + "\n\n"
		s += helperStyle.Render("Enter JSON property (my.property.key)") + "\n\n"
	}

	if m.state == preferredStatusView {
		s += "Preferred Status: \n\n"
		s += m.textinput.View() + "\n\n"
		s += helperStyle.Render("Enter preferred HTTP status (100-599)") + "\n\n"
	}

	if m.state == insecureSkipVerifyView {
		s += "Insecure Skip Verify: \n\n"
		s += m.textinput.View() + "\n\n"
		s += helperStyle.Render("Enter true/false") + "\n\n"
	}

	if m.errorMsg != "" {
		s += errorMessageStyle.Render(m.errorMsg) + "\n\n"
	}

	if m.state == nameView {
		s += faint.Render("enter = save | esc = cancel")
	} else if m.state != listView {
		s += faint.Render("enter = continue | esc = go back")
	}

	if m.state == listView {
		for i, o := range m.services {
			prefix := " "
			if i == m.listIndex {
				prefix = ">"
			}
			shortEndpoint := strings.ReplaceAll(o.Endpoint, "\n", " ")
			if len(shortEndpoint) > 60 {
				shortEndpoint = shortEndpoint[:60] + "..."
			}
			s += listEnumeratorStyle.Render(prefix) + o.Name + " | " + faint.Render(shortEndpoint) + "\n\n"
			if o.LastStatusInfo == "" {
				s += "Waiting" + m.spinner.View() + "\n\n"
			} else {
				s += o.LastStatusInfo + " " + m.pulseSpinner.View() + "\n\n"
			}
		}
		s += faint.Render("n - new service | q - quit | d - delete | r - restart history")
	}

	s += "\n\n" + helperStyle.Render("goardian v1.0.0 by DavidArtifacts")
	s += "\n\n" + helperStyle.Render("GitHub: https://github.com/DigitalArtifactory/goardian") + "\n\n"
	return s
}

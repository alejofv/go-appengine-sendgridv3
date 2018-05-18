package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
)

var (
	pageTemplate = template.Must(template.ParseFiles("index.html"))
)

type pageTemplateParams struct {
	Confirmation string
	Warning      string
	Form         bool
}

type emailContactParams struct {
	Name    string
	Email   string
	Subject string
	Message string
}

func main() {
	http.HandleFunc("/", pageHandler)

	appengine.Main()
}

func pageHandler(w http.ResponseWriter, r *http.Request) {
	pageParams := pageTemplateParams{
		Form: true,
	}

	if r.Method == "GET" {
		pageTemplate.Execute(w, pageParams)
		return
	}

	name := r.FormValue("name")
	email := r.FormValue("email")
	subject := r.FormValue("subject")

	if name == "" || email == "" || subject == "" {
		w.WriteHeader(http.StatusBadRequest)

		pageParams.Warning = "Please verify required fields"
		pageTemplate.Execute(w, pageParams)
		return
	}

	contactParams := emailContactParams{
		Email:   email,
		Name:    name,
		Subject: subject,
		Message: r.FormValue("message"),
	}

	ctx := appengine.NewContext(r)
	httpClient := urlfetch.Client(ctx)

	res, err := sendEmail(httpClient, contactParams)

	if isOk(res) {
		pageParams.Confirmation = fmt.Sprintf("Thanks for your message %s!", name)
		pageParams.Form = false
	} else {
		fmt.Printf("%v: %v", res, err)
		pageParams.Warning = "Error sending message."
	}

	pageTemplate.Execute(w, pageParams)
}

func sendEmail(httpClient *http.Client, params emailContactParams) (*rest.Response, error) {
	// v3 API

	fromName := os.Getenv("SENDGRID_FROM_NAME")
	fromEmail := os.Getenv("SENDGRID_FROM_EMAIL")
	templateID := os.Getenv("SENDGRID_TEMPLATE_ID")
	apiKey := os.Getenv("SENDGRID_API_KEY")

	from := mail.NewEmail(fromName, fromEmail)
	to := mail.NewEmail(params.Name, params.Email)

	plainText := mail.NewContent("text/plain", params.Subject)
	html := mail.NewContent("text/html", params.Subject)

	message := new(mail.SGMailV3)
	message.Subject = "Test mail from AppEngine"
	message.SetFrom(from)
	message.SetTemplateID(templateID)
	message.AddContent(plainText, html)

	p := mail.NewPersonalization()
	p.AddTos(to)
	p.SetSubstitution("-contact_name-", params.Name)
	p.SetSubstitution("-contact_email-", params.Email)
	p.SetSubstitution("-contact_subject-", params.Subject)
	p.SetSubstitution("-contact_message-", params.Message)

	message.AddPersonalizations(p)

	sendgrid.DefaultClient.HTTPClient = httpClient

	client := sendgrid.NewSendClient(apiKey)
	response, err := client.Send(message)

	return response, err
}

func isOk(response *rest.Response) bool {
	return response.StatusCode >= 200 && response.StatusCode < 300
}

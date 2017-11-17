package users

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/rivo/sessions"
)

var (
	// The list of htmlTemplates that have already been parsed.
	htmlTemplates      map[string]*template.Template
	htmlTemplatesMutex sync.Mutex
)

// retrieveTemplate returns an HTML template with the given filename (located in
// Config.HTMLTemplateDir or a subdirectory of it, depending on the value of
// Config.Internationalization), retrieving it from the cache if
// Config.CacheTemplates is true and if it has been loaded before. The template
// will include all templates specified in Config.HTMLTemplateIncludes.
func retrieveTemplate(request *http.Request, htmlTemplate string) (tmpl *template.Template, err error) {
	// Determine template subdirectory.
	var subdirectory string
	if Config.Internationalization {
		if cookie, e := request.Cookie("lang"); e == nil {
			languageFormat := regexp.MustCompile("^[a-zA-Z]{2}(-[a-zA-Z]{2})?$")
			if languageFormat.MatchString(cookie.Value) {
				if dir, er := os.Stat(filepath.Join(Config.HTMLTemplateDir, cookie.Value)); er == nil {
					if dir.IsDir() {
						subdirectory = cookie.Value
					}
				}
			}
		}
	}

	var ok bool
	if Config.CacheTemplates {
		// Query the cache first.
		htmlTemplatesMutex.Lock()
		defer htmlTemplatesMutex.Unlock()
		if htmlTemplates == nil {
			htmlTemplates = make(map[string]*template.Template)
		}
		tmpl, ok = htmlTemplates[subdirectory+"/"+htmlTemplate]
	}

	if !ok {
		// Create a new template with functions.
		tmpl = template.New(htmlTemplate).Funcs(template.FuncMap{
			"title": func(values ...interface{}) interface{} {
				// Add a title to a map.
				if len(values) == 0 {
					return nil
				}
				if len(values) == 1 {
					return values[0]
				}
				m, ok := values[0].(map[string]interface{})
				if !ok {
					return values[0]
				}
				m["title"] = values[1]
				return m
			},
		})

		// Load template and includes.
		fileList := append([]string{htmlTemplate}, Config.HTMLTemplateIncludes...)
		for index, file := range fileList {
			fileList[index] = filepath.Join(Config.HTMLTemplateDir, subdirectory, file)
		}
		tmpl, err = tmpl.ParseFiles(fileList...)
		if err != nil {
			return
		}
	}

	// Store in cache if required.
	if Config.CacheTemplates {
		htmlTemplates[subdirectory+"/"+htmlTemplate] = tmpl
	}

	return
}

// RenderPage renders the HTML template with the given name (located in
// Config.HTMLTemplateDir or a subdirectory of it, depending on the value of
// Config.Internationalization), attached to the given data, and sends it to the
// browser. It also instructs the browser not to cache this page. Other
// templates used by this htmlTemplate must be specified in
// Config.HTMLTemplateIncludes (with the exception of error and message
// templates which are included automatically).
func RenderPage(response http.ResponseWriter, request *http.Request, htmlTemplate string, data interface{}) {
	// This is a simple version of RenderProgramError(), used here to avoid
	// an endless recursion.
	programError := func(response http.ResponseWriter, internalMessage, externalMessage string, err error) {
		response.WriteHeader(http.StatusInternalServerError)
		errorID, e := sessions.RandomID(8)
		if e != nil {
			errorID = "noid"
		}
		Config.Log.Printf("%s: %s: %s", errorID, internalMessage, err)
		fmt.Fprintf(response, "%s (%s)", externalMessage, errorID)
	}

	// Get the template.
	tmpl, err := retrieveTemplate(request, htmlTemplate)
	if err != nil {
		programError(response, fmt.Sprintf(`Could not retrieve template "%s"`, htmlTemplate), "Could not retrieve template", err)
		return
	}

	// Execute the template and send it to the browser.
	response.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	response.Header().Set("Pragma", "no-cache")
	response.Header().Set("Expires", "0")
	if err := tmpl.Execute(response, data); err != nil {
		programError(response, fmt.Sprintf(`Template "%s" could not be executed`, htmlTemplate), "Could not execute template", err)
	}
}

// RenderProgramError outputs a program error using the programerror.gohtml
// template and logs it along with a reference code. This should only be called
// on an unexpected error that we cannot recover from. An internal server error
// code is sent. If the external message is the empty string, the internal
// message is used.
func RenderProgramError(response http.ResponseWriter, request *http.Request, internalMessage, externalMessage string, err error) {
	response.WriteHeader(http.StatusInternalServerError)
	errorID, e := sessions.RandomID(8)
	if e != nil {
		errorID = "noid"
	}
	if err == nil {
		err = errors.New("No error message provided")
	}
	Config.Log.Printf("Program error %s: %s: %s", errorID, internalMessage, err)
	if externalMessage == "" {
		externalMessage = internalMessage
	}
	RenderPage(response, request, "programerror.gohtml", fmt.Sprintf("%s (%s)", externalMessage, errorID))
}

// RenderPageBasic calls RenderPage() on a map with a "config" key mapped to the
// Config object and, if the user is logged in, a "user" key mapped to the
// provided user (which can be nil if no user is logged in).
func RenderPageBasic(response http.ResponseWriter, request *http.Request, htmlTemplate string, user User) {
	data := map[string]interface{}{"config": Config}
	if user != nil {
		data["user"] = user
	}
	RenderPage(response, request, htmlTemplate, data)
}

// RenderPageError calls RenderPage() on a map with a "config" key mapped to the
// Config object, an "error" key mapped to the output of an error template
// (whose filename is constructed by prefixing errorName with "error_" and
// suffixing it with ".gohtml"), an "infos" key mapped to errorInfos, and - if a
// user was provided, meaning a user is logged in - a "user" key mapped to that
// user. The error template is only executed on errorInfos. The function also
// sends a Bad Request HTTP header.
func RenderPageError(response http.ResponseWriter, request *http.Request, htmlTemplate, errorName string, errorInfos interface{}, user User) {
	// Render the error message template first.
	errTmpl, err := retrieveTemplate(request, fmt.Sprintf("error_%s.gohtml", errorName))
	if err != nil {
		RenderProgramError(response, request, "Could not retrieve error template: "+errorName, "Could not retrieve error template", err)
		return
	}
	var errMsg bytes.Buffer
	if err = errTmpl.Execute(&errMsg, errorInfos); err != nil {
		RenderProgramError(response, request, "Could not execute error template: "+errorName, "Could not execute error template", err)
		return
	}

	response.WriteHeader(http.StatusBadRequest)
	data := map[string]interface{}{
		"config": Config,
		"error":  template.HTML(strings.TrimSpace(string(errMsg.Bytes()))),
		"infos":  errorInfos,
	}
	if user != nil {
		data["user"] = user
	}
	RenderPage(response, request, htmlTemplate, data)
}

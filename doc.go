/*
Package users implements common web user workflows. Most of the provided
functions are regular net/http handler functions. The following functionality
is provided:

  - Signing up for a new user account
  - Logging in and out
  - Checking login status
  - Resetting forgotten passwords
  - Changing email and password

Special emphasis is placed on reducing the risk of someone hijacking user
accounts. This is achieved by enforcing a certain user structure and following
certain procedures:

  - Users are identified by their email address.
  - New or changed email addresses must be verified by clicking on a link
    emailed to that address.
  - Users authenticate by entering their email address and a password.
  - Password integrity is checked via ReasonablePassword() in the package
    github.com/rivo/sessions.
  - Forgotten passwords are reset by clicking on a link emailed to the user.

If your application does not follow these principles, you may not be able to
use this package as is. However, the code may serve as a starting point if you
apply its principles to your own use case.

Note also the following:

  - The package's functionality is based on the sessions package
    github.com/rivo/sessions.
  - You need access to an SMTP email server to send verification and password
    reset emails.
  - It uses Golang HTML templates for the various pages and text templates for
    the various emails sent to the user. These may all be customized to your
    needs.
  - The template rendering functions used in this package are public and may
    prove useful for other parts of your application.
  - The package is database agnostic. You may choose any storage system to store
    and retrieve user data.
  - Internationalization is supported via the "lang" browser cookie.

Basic Example

The users.Main() function registers all handlers and starts an HTTP server:

  if err := users.Main(); err != nil {
    panic(err)
  }

Any other handlers can be added to the http.DefaultServeMux before calling
users.Main(). Alternatively, you can start your own HTTP server. See the
implementation of users.Main() for how to add the package's handlers.

Package Configuration

See the package example for a most basic way to use the package. In addition,
the global Config struct contains all the variables that need to be adjusted for
your specific application. It provides sensible defaults out of the box which
you can see in its documentation. The fields are as follows:

  - ServerAddr: The address the HTTP server binds to. This is only needed if
    you start the server using the package's Main() function.
  - Log: A logger for all major events of the package.
  - LoggedIn: A function which is called any time a user was logged in
    successfully. This may be used for example to record the login time.
  - NewUser: A function which returns a new object that implements the User
    interface.
  - PasswordNames: A list of strings to be excluded from passwords. This is
    usually the application name or the domain name. Anything specific to the
    application. It will be passed to ReasonablePassword() in the package
    github.com/rivo/sessions.
  - Route*: The fields starting with "Route" contain the routes for the various
    pages. They are used throughout the package's code as well as in the
    templates.

The following fields control how templates are handled:

  - CacheTemplates: Whether or not templates are cached. If set to true,
    templates are only loaded the first time they are used and then stored for
    successive uses. This reduces the load on the local hard drive but any
    changes after the first use will not become visible.
  - HTMLTemplateDir: The directory where the HTML templates are located.
  - HTMLTemplateIncludes: Because Golang requires any referenced templates to
    be included while parsing, if you need to include more templates than the
    default "header.gohtml" and "footer.gohtml", they need to be specified here.
  - MailTemplateDir: The directory where the email templates are located.
  - MailTemplateIncludes: Same as HTMLTemplateIncludes but for email templates.

If your application supports internationalization, you can set the
Internationalization field to true. If set to true, this package's code checks
for the "lang" cookie and appends its value to the HTMLTemplateDir and
MailTemplateDir directories to search for template files. Cookie values must be
of the format "xx" or "xx-XX" (e.g. "en-US"). If they don't have this format or
if the corresponding subdirectory does not exist, the search falls back to the
HTMLTemplateDir and MailTemplateDir directories. It is up to the application to
set the "lang" cookie.

Emails are sent if the SendEmails field is set to true. You can provide your
own email function by implementing the SendEmail field. Alternatively, the
net/smtp package is used to send emails. The following fields need to specified
(fields starting with "SMTP" are only needed when you don't provide your own
SendEmail implementation):

  - SenderName: The name to be shown in the email's "From" field.
  - SenderEmail: The sender's email address.
  - SMTPHostname: The mail server's host address.
  - SMTPPort: The mail server's port.
  - SMTPUsername: The username to authenticate with the mail server.
  - SMTPPassword: The password to authenticate with the mail server.

A number of functions serve as the interface to your database:

  - SaveNewUserAtomic: Saves a new user to the database after making sure that
    no such user existed before.
  - UpdateUser: Updates an existing user.
  - LoadUserByVerificationID: Loads a user given a verification ID.
  - LoadUserByPasswordToken: Loads a user given a password reset token.
  - LoadUserByEmail: Loads a user given an email.

The User Object

Anyone using this package must define a type which implements this package's
User interface. A user is in one of three possible states:

  - StateCreated: The user exists but has not yet been verified and can
    therefore not yet use the application.
  - StateVerified: The user has been verified and has access to the application.
  - StateExpired: The user account has expired. The application cannot be used
    anymore.

Users have an ID which must be unique (e.g. generated by CUID() in the package
github.com/rivo/sessions). But this package may access users based on their
unique email address, their verification ID, or their password reset token.

You must implement the Config.NewUser function.

Template Structure and Functions

There are basic HTML templates (in the "html" subdirectory) and email templates
(in the "mail" subdirectory). All HTML templates starting with "error_" are
templates that will generate error messages which are then embedded in another
HTML template. When starting to work with this package, you will want to make
a copy of these two subdirectories and modify the templates to your needs.

This package implements some functions to render templates which are also public
so you may use them in other places, too. The function RenderPage() takes a
template filename and a data object (to which the template will be bound),
renders the template, and sends it to the browser. Instead of calling this
function, however, RenderPageBasic() is used more often. It calls RenderPage()
but populates the data object with the Config object and the User object (if one
was provided).

If an error message needs to be shown to the user, RenderPageError() can be
used. This actually involves two templates, one to generate only the error
message (these template files start with "error_"), and the other to generate
the HTML file which shows the error message. Config and User will also be bound
to the latter as well as any data sent to the error message template.

There is another function for errors, RenderProgramError(), which is used to
show program errors. These are unexpected errors, for example database
connection issues, and should always be followed up on. While the user usually
only sees a basic error message, more detailed information about the error is
sent to the logger for further inspection.

The SendMail() function renders mail templates (based on text/template) to send
them to the user's email address.

When writing your own templates, it is helpful to make a copy of the existing
example templates and modify them to your needs.

All templates include a header and a footer file. If you include more files,
you will need to set the Config.HTMLTemplateIncludes and
Config.MailTemplateIncludes fields accordingly.
*/
package users

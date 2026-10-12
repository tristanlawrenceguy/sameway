package inbox

import "github.com/tristanlawrenceguy/sameway/internal/web"

// Routes are the inbox's addresses, which the server adds to its one route
// table (server/routes.go).
var Routes = []web.Route[*Service]{
	{Pattern: "GET /mail", Handle: (*Service).mailPage, Access: web.Owner},
	{Pattern: "GET /mail/contact.vcf", Handle: (*Service).mailContact, Access: web.Owner},
	{Pattern: "POST /mail/connect", Handle: (*Service).mailConnect, Access: web.Owner, Persons: "a mailbox and its password are its owner's to give", Reach: web.Outward},
	{Pattern: "POST /mail/off", Handle: (*Service).mailOff, Access: web.Owner, Persons: "a mailbox and its password are its owner's to give", Reach: web.Outward},
	{Pattern: "POST /mail/task", Handle: (*Service).mailTask, Access: web.People, Tool: "create_record", Reach: web.Inward},
	{Pattern: "POST /mail/sorted", Handle: (*Service).mailSorted, Access: web.People, Tool: "update_record", Reach: web.Inward},
	{Pattern: "POST /mail/answered", Handle: (*Service).mailAnswered, Access: web.People, Tool: "update_record", Reach: web.Inward},
	{Pattern: "POST /tags/keep", Handle: (*Service).tagKeep, Access: web.People, Persons: "keeping a tag is the person's judgement, which the action then follows", Reach: web.Inward},
	{Pattern: "POST /tags/off", Handle: (*Service).tagOff, Access: web.People, Tool: "update_record", Reach: web.Inward},
	{Pattern: "POST /tags/change", Handle: (*Service).tagChange, Access: web.People, Tool: "update_record", Reach: web.Inward},
	{Pattern: "POST /suggested/yes", Handle: (*Service).suggestedYes, Access: web.People, Tool: "create_record", Reach: web.Inward},
	{Pattern: "POST /suggested/change", Handle: (*Service).suggestedChange, Access: web.People, Tool: "create_record", Reach: web.Inward},
	{Pattern: "POST /suggested/no", Handle: (*Service).suggestedNo, Access: web.People, Persons: "turning a suggestion down is the person's answer, which the action then follows", Reach: web.Inward},
}

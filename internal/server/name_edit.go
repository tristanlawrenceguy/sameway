package server

import "html/template"

// A person renames a list, a calendar or a chart where it is. Their name
// is a prop the page fills from the records around it (a chart's caption
// is written for it when left out), so it is not marked where it shows;
// and a collection's cards carry the card's own marks, which are not the
// block's. So the block says what can be edited in the template the
// inline editor reads first (10-edit-fields.js): its name alone, as Name.
// Edit then opens one field, Save posts prop-<name> to the block's props,
// and the change is logged and undone like any other. In the agent
// evaluation (T5, "call it Up next") a person-shaped agent looked for a
// way to rename the Tasks list, found none, and said it could not be done.

// nameProps is the prop that names each kind of data-bound block.
var nameProps = map[string]string{collectionComponent: "label", calendarComponent: "caption", chartComponent: "caption"}

// nameEdit is the block's name, marked for the inline editor, for those
// who may change it.
func nameEdit(component string, props map[string]any, convo *conversation) template.HTML {
	prop, ok := nameProps[component]
	if !ok || convo == nil || convo.LookOnly {
		return ""
	}
	v, _ := props[prop].(string)
	return template.HTML(`<template data-edit-fields><span data-prop="` + prop + `" data-label="Name" data-source="` + template.HTMLEscapeString(v) + `"></span></template>`)
}

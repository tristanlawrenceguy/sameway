package bench

import "time"

// The messages sorting is measured on: what comes to a person (bookings,
// bills, school, work, friends asking, and the newsletters and receipts
// that ask nothing), each with what is right for it.

type triageCase struct {
	name, text string
	task       bool
	due        string // YYYY-MM-DD, or "" when none is said
	important  bool
}

// triageSent is when every message was sent: Monday 12 October 2026, 09:00.
var triageSent = time.Date(2026, 10, 12, 9, 0, 0, 0, time.Local)

var triageCases = []triageCase{
	{"school form", "From: Oakfield School\nSubject: Trip on Friday\n\nDear parents, please return the signed consent form for the museum trip by Friday.", true, "2026-10-16", true},
	{"water bill", "From: City Water\nSubject: Your bill is ready\n\nYour bill of £48.20 is due on 26 October 2026. Pay online to avoid a late fee.", true, "2026-10-26", true},
	{"dentist booking", "From: Smile Dental\nSubject: Appointment confirmed\n\nYour appointment is on Thursday 15 October at 14:30. Reply C to cancel.", true, "2026-10-15", true},
	{"newsletter", "From: Garden Co\nSubject: Autumn bulbs are here\n\nOur autumn range has arrived. 20% off tulips this week only.", false, "", false},
	{"receipt paid", "From: Shop\nSubject: Your receipt\n\nThank you for your order. £12.99 paid with Visa ending 4421. Nothing more to do.", false, "", false},
	{"friend favour", "Hi! Could you lend me your drill this weekend? I'll pick it up Saturday morning if that works.", true, "2026-10-17", false},
	{"passport", "From: HM Passport Office\nSubject: Your passport expires soon\n\nYour passport expires on 3 December 2026. Renew it now to travel.", true, "2026-12-03", true},
	{"tax", "From: Revenue\nSubject: Self assessment\n\nYour tax return must be filed by 31 January 2027.", true, "2027-01-31", true},
	{"meeting moved", "From: Priya (work)\nSubject: Standup moved\n\nTomorrow's standup is moved to 10:30, same room.", true, "2026-10-13", false},
	{"parcel delivered", "From: Couriers\nSubject: Delivered\n\nYour parcel was delivered to your front door at 11:02.", false, "", false},
	{"reply waiting", "From: Joe (client)\nSubject: Quote?\n\nHi, did you get a chance to look at the quote? I need an answer by Wednesday to book the builders.", true, "2026-10-14", true},
	{"birthday party", "From: Mum\nSubject: Saturday\n\nDon't forget Gran's 80th on Saturday at 3pm, bring a card!", true, "2026-10-17", false},
	{"advert", "From: Phone Deals\nSubject: Upgrade today\n\nGet the new phone for £0 upfront. Offer ends Sunday.", false, "", false},
	{"car insurance", "From: Insurer\nSubject: Renewal\n\nYour car insurance renews on 1 November. Check your details and renew before then or you will not be covered.", true, "2026-11-01", true},
	{"library", "From: Library\nSubject: Books due\n\nTwo books are due back on 19 October. Renew online.", true, "2026-10-19", false},
	{"shopping list", "Can you get milk, eggs and bread on the way home?", true, "", false},
	{"invoice to send", "From: Ana\nSubject: Invoice\n\nCould you send me the invoice for September when you can? Thanks!", true, "", true},
	{"webinar recording", "From: Events\nSubject: Recording available\n\nThe recording of last week's webinar is now available to watch any time.", false, "", false},
	{"vet", "From: Vets4Pets\nSubject: Booster due\n\nRex's annual booster is due. Please book an appointment in the next two weeks.", true, "", true},
	{"password reset", "From: Bank\nSubject: Password changed\n\nYour password was changed. If this was you, no action is needed.", false, "", false},
	{"rent", "From: Landlord\nSubject: Rent\n\nJust a reminder that rent is due on the 1st as usual.", true, "2026-11-01", true},
	{"group chat plan", "Dinner at ours next Friday at 7, can you bring dessert?", true, "2026-10-23", false},
	{"doctor results", "From: Surgery\nSubject: Test results\n\nYour test results are back. Please call the surgery to discuss them.", true, "", true},
	{"social notification", "From: Social\nSubject: 5 people liked your photo\n\nSee who liked your photo.", false, "", false},
	{"work deadline", "From: Boss\nSubject: Report\n\nCan you have the quarterly report to me by end of day Thursday?", true, "2026-10-15", true},
	{"order shipped", "From: Store\nSubject: Shipped\n\nYour order has shipped and will arrive Wednesday.", false, "", false},
	{"neighbour", "Hi, it's Sam next door. We're away from tomorrow for a week, could you water the plants?", true, "2026-10-13", false},
	{"council", "From: Council\nSubject: Bin collection change\n\nFrom next week, recycling is collected on Tuesdays instead of Mondays.", false, "", false},
	{"concert tickets", "From: Tickets\nSubject: Your tickets\n\nYour e-tickets for Saturday 24 October are attached. Show them at the door.", false, "", false},
	{"sign lease", "From: Agency\nSubject: Lease ready\n\nThe lease is ready to sign. Please sign it by 20 October so we can confirm the move.", true, "2026-10-20", true},
}

# Alert

This context owns how a shop warns the customers who are present in it right now about an emergency, and how those customers come to be known to the shop in the first place. It is the glossary for the Alert product (`alert-api`, `alert-web`); identity terms (User, Client, System, Role) come from UM and are not redefined here.

## Language

### People in the shop

**Check-in**:
One customer's declared presence at a Branch for a limited time: who to reach (a mobile phone), how many came, which table, which language, and what they consented to. A Check-in is the unit an alert is addressed to. It ends when the customer checks out, when staff check them out, or when it expires.
_Avoid_: Visit, registration, guest, session

**Recipient**:
A Check-in that an alert may be sent to at the moment of sending: still present, confirmed or exempt from confirmation, with at least one channel to reach. Only Recipients count toward an Emergency Event.
_Avoid_: Subscriber, contact, audience

**Confirmation**:
The step that proves a Check-in's phone is real before it can receive alerts: a one-time code sent to the phone and typed back, identified to the customer by a short reference code. A Branch may waive it (skip OTP), in which case a Check-in is live as soon as consent is given.
_Avoid_: OTP (the code is one part of the Confirmation), verification, activation

**Consent**:
The customer's acceptance of the privacy notice at Check-in, recorded with the notice version and time; marketing consent is a separate, optional choice. Withdrawing Consent ends the Check-in.
_Avoid_: Agreement, opt-in (for the privacy notice)

**Retention**:
How long a Branch keeps a Check-in and its delivery records after they end, at most 24 hours, after which they are deleted automatically.
_Avoid_: TTL, expiry window

**Entry point**:
How a customer reaches the Check-in form for a Branch: a QR code that staff issue for a table and can revoke, or a direct link that names the Branch. Both lead to the same Check-in.
_Avoid_: QR token (one kind of Entry point), invite

### Alerting

**Emergency Event**:
One alert sent by a staff member to every Recipient of a Branch, of one Event Type, recorded with who triggered it, how it was confirmed, how many were reached and what each channel achieved. It is open until staff close it.
_Avoid_: Alert (the message), notification, broadcast, incident

**Event Type**:
What the alert is about: fire, evacuate, avoid an area, suspicious object, brawl, external threat, all clear, or test. All clear and test are the two that bypass the Cooldown.
_Avoid_: Category, severity, alert kind

**Trigger confirmation**:
The deliberate act that stands between pressing the button and sending: holding the button for three seconds or entering the Branch PIN, as the Branch's settings require.
_Avoid_: Two-step, double confirm

**Cooldown**:
The minimum time a Branch waits between two Emergency Events so one incident does not become a flood of messages. An admin may override it, and the override is recorded on the event.
_Avoid_: Rate limit, throttle, debounce

**Channel**:
A way to reach a Recipient: web push, SMS, or LINE. Web push is always on; SMS and LINE are switched on per Client because they cost money per message.
_Avoid_: Provider (the vendor behind a Channel), medium, transport

**Delivery**:
One message to one Recipient over one Channel and what became of it: queued, sent, delivered, or failed, with the vendor's own status when it reports back.
_Avoid_: Send, dispatch log, receipt

**Message Template**:
The wording of an alert for one Event Type, kept per Client in Thai and English, with optional per-Channel variants. A Recipient receives the text in their preferred language.
_Avoid_: Notification text, copy, string

### Staff and the shop

**Branch**:
One physical location of a Client with its own settings: shop name, Retention, Cooldown, Trigger confirmation method, whether Confirmation may be skipped, and the alert PIN. Check-ins and Emergency Events belong to a Branch.
_Avoid_: Store, location, site, tenant

**Alert Permission**:
What one staff member may trigger at a Branch: the Event Types they are allowed to send and whether they receive test alerts. Granted per Client on top of the UM Role.
_Avoid_: Staff permission, role (the UM word), ACL

**Messaging Config**:
A Client's own SMS gateway and LINE account used to send over those Channels, with the switches that enable each. Where a Client has not set one, the platform default is used.
_Avoid_: Provider settings, integration, credentials

**Audit entry**:
The record that a staff member did something that matters: triggered or tested an alert, checked a customer out, changed a template, PIN, setting, permission or QR, overrode a Cooldown, exported a report, or withdrew consent on a customer's behalf.
_Avoid_: Log line, history, activity

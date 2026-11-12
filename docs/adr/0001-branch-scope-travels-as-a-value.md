# Branch scope travels as a value, not as a string

A staff request used to carry its Branch as a gin string key set by `RequireBranch`, and the repositories took a loose `clientId` with an optional `branchId`. Two reads by id (`Checkout`, `GetEventById`) filtered by `_id` alone, so a staff member of one Branch could check out a Check-in, or read an Emergency Event, of another Branch of the same Client (fixed in v1.4.1). Nothing in the types made the omission visible.

Now the Branch is a value at both seams:

- `RequireBranch` resolves one `Staff` (UM User and Role, the Permission's Branch as a `Scope`, and the Event Types it allows). Handlers read `staff.Current(ctx)`; the gin keys `BranchId` and `AllowedEventTypes` no longer exist. `Staff` owns the three permission rules: `MayTrigger`, `MayTest`, `MayOverrideCooldown`. Route-level UM Role checks (`Auth.AtLeast`) stay where they are (UM ADR-0005).
- Every Branch-scoped method of `ICheckIn` and `IEmergencyEvent` takes a `repositories.Scope{ClientId, BranchId}` and filters on both. A foreign-Branch single read is `mongo.ErrNoDocuments`; a foreign-Branch list or count is empty.

There is no default Branch (CONTEXT: Alert Permission). A customer is never Staff: after Confirmation a customer request acts through the Scope of the Check-in it loaded; before Confirmation (`verify`, `resend-otp`) the request carries only the tenant ref, so `FindCheckInByRef` stays Client-scoped by nature. Audit entries and Deliveries record the Branch as data; they are not filtered by it here.

# Branch scope travels as a value, not as a string

A staff request used to carry its Branch as a gin string key set by `RequireBranch`, and the repositories took a loose `clientId` with an optional `branchId`. Two reads by id (`Checkout`, `GetEventById`) filtered by `_id` alone, so a staff member of one Branch could check out a Check-in, or read an Emergency Event, of another Branch of the same Client (fixed in v1.4.1). Nothing in the types made the omission visible.

Now the Branch is a value at both seams:

- `RequireBranch` resolves one `Staff` (UM User and Role, the Permission's Branch as a `Scope`, and the Event Types it allows). Handlers read `staff.Current(ctx)`; the gin keys `BranchId` and `AllowedEventTypes` no longer exist. `Staff` owns the three permission rules: `MayTrigger`, `MayTest`, `MayOverrideCooldown`. Route-level UM Role checks (`Auth.AtLeast`) stay where they are (UM ADR-0005).
- Every Branch-scoped method of `ICheckIn` and `IEmergencyEvent` takes a `repositories.Scope{ClientId, BranchId}`. The Client is enforced by which database the method opens (`collection(scope.ClientId)`: one database per Client); the Branch is enforced by a `branchId` filter that every Branch-scoped method applies, reads and writes alike. A foreign-Branch single read is `mongo.ErrNoDocuments`; a foreign-Branch list or count is empty; a foreign-Branch write changes nothing (`Checkout` reports `mongo.ErrNoDocuments`, the other writes report no error).

There is no default Branch (CONTEXT: Alert Permission). A customer is never Staff: after Confirmation a customer request acts through the Scope of the Check-in it loaded; before Confirmation (`verify`, `resend-otp`) the request carries only the tenant ref, so `FindCheckInByRef` stays Client-scoped by nature. `Scope` does not cover everything. Branch Settings, Alert Permissions (including test recipients), QR tokens, message templates, OTP requests and Deliveries still take loose `clientId` and `branchId` or id arguments. Audit entries and Deliveries record the Branch as data; they are not filtered by it here.

QR management (`/admin/qr`: list, create, revoke, image) is ADMIN-only and Client-wide. An ADMIN lists and creates QR tokens for any Branch of the Client (the Branch is a query or body field), and revoke and image look a token up by id within the Client. The route group does sit behind `RequireBranch`, so the ADMIN must hold an Alert Permission, but the Branch it names is not used to filter QR. This is left as it is on purpose; whether QR should become Branch-scoped is an open product decision.

The one deliberate exception above, `FindCheckInByRef` before Confirmation, stays as written.

## What is babyfilecab? 

BabyFileCab is a digital filing cabinet and workflow tool built specifically for tax and accounting practices.

BabyFileCab is a local-first desktop document management system designed specifically for tax and accounting professionals. It gives tax preparers, Enrolled Agents, CPAs, bookkeepers, and small accounting firms a structured digital filing cabinet for organizing client records by client, tax year, section, and permanent folder. Instead of keeping tax documents scattered across Windows folders, email downloads, spreadsheets, and PDFs, BabyFileCab brings them into one organized workspace.

BabyFileCab can be used to manage and preview documents, search across client files, maintain client notes, track activity, and organize workflow. It also includes accounting-practice tools such as a firm calendar, federal tax calculator, engagement letter generator, invoice generator, IRS yearly-average currency converter, and audit trail. A major part of the design is that it is local-first. Client documents are stored on the user's own computer rather than requiring them to be uploaded to BabyFileCab-operated cloud storage. Its security architecture includes features such as AES-256-GCM encryption, company-specific vault keys, Argon2id password protection, encrypted backups, automatic locking, and protected audit history.

Its goal is to combine document organization, client management, security, and everyday tax-office tools in one easy-to-use desktop application.

**8879 Sent**, **Waiting for Signature**, **Signed** and **Ready to E-file** are available both by right-clicking a scheduled client and in the search panel's Change Assignment dropdown. The chosen label appears beside the client's name in the weekly calendar and in the scheduled-date selector. These are manually selected workflow labels; choosing them does not send a form, collect a signature or file a return. Each scheduled entry has one status, so selecting a new status replaces the previous one. Clear Status removes it. The statuses persist in encrypted company calendar metadata.

JavaScript syntax and eight Node test files pass, including labels for all four new statuses. Go validation tests include the new values but require a local Go toolchain to run. Native desktop visual verification remains local.

## Go to Client on Calendar from search

Every scheduled client in All Clients search results now has **Go to Client on Calendar** buttons showing its scheduled date and preparer. Multiple scheduled dates appear separately so you can choose the right entry. Clicking jumps to its week, switches the preparer filter when necessary, clears search to reveal the calendar, and opens the client's document details. Clients without a date show Not scheduled yet and retain Assign Preparer & Date.

JavaScript syntax and the eight existing Node test files pass. Native desktop visual verification remains to be performed locally.

## Main-page calendar shortcut and all-client scheduling

Click the **📅 Firm Calendar** button beside Open Data Folder on the main page. Calendar opening refreshes the company document tree. Search now includes **All Clients**, matching every active client in that tree by name or ID, including clients with no uploaded documents or calendar assignments. This list is independent of the selected preparer view. Scheduled matches remain below it and honor the selected preparer view.

Click **Assign Preparer & Date** beside a client, choose a 2026 date and a registered company preparer (or Unassigned), and click Assign. The calendar jumps to the chosen week and preparer view. If that client is already on that date, confirmation is required before changing its preparer; the existing status is preserved. Archived clients are excluded until restored to the document tree. Assignment metadata continues to use the company's encrypted SQLite storage.

JavaScript syntax and all eight Node test files pass, including unscheduled clients, ID/name matching, case handling, and file-node exclusion. Native desktop visual testing remains to be performed locally.

## Larger invoice and firm/preparer calendar views

Create an Invoice now opens at the same nearly full-window size as the Tax Calculator; an overriding rebuilt-form height cap has been removed for this dialog.

The Firm Calendar's top-right initials button opens a dropdown containing Entire Firm, Unassigned Clients and every registered user within this company. Whole-firm views display preparer names with scheduled clients; individual views and year-wide search filter to that preparer's assignments. Every registered company user is available as a preparer because this version has Staff/Administrator roles rather than a separate tax-preparer role. Calendar views are workload filters, not document-access restrictions.

Right-click a scheduled client (including a search result), choose Reassign To Preparer, select a user and save. This changes the owner of that specific scheduled entry; its date, status and other scheduled entries stay intact. Rescheduling to a different date retains its preparer. Existing entries remain Unassigned until deliberately assigned. New entries in an individual view belong to that preparer; new entries in Entire Firm belong to the signed-in user. The existing date reassignment option remains available separately.

Preparer usernames persist inside the company's encrypted SQLite calendar metadata. Backend validation rejects unknown users and users from other companies. JavaScript syntax and seven Node test files pass. Added Go tests cover company membership, expired sessions, targeted reassignment, status preservation and serialization. Go/native desktop verification remains to be performed locally.

## Engagement Letter Generator opening-size fix

The Engagement Letter Generator now uses the same viewport width and height as the Federal Tax Calculator, with 16 pixels of margin on each edge. A specific rebuilt-form selector previously overrode the larger height with a 900-pixel cap; the scoped override removes that cap while preserving the form's internal scrolling and footer buttons. This fix targets the engagement letter window only.

CSS selector precedence was inspected. Native desktop visual verification remains to be performed locally.

## Top Search menu and larger tool windows

Search is now a top-menu dropdown between **File** and **Edit**, with **Search Documents** and **Search Clients**. The action-toolbar Search button has been removed. Document search opens the existing company document search; client search opens the searchable active-client list and selects the chosen client in the workspace, including clients without uploaded documents.

The Federal Tax Calculator, Engagement Letter Generator, Create Invoice, Currency Converter and Audit Trail open nearly full-window with internal scrolling. Existing invoice single-scrollbar behavior is retained. An incorrectly escaped engagement-letter CSS block was repaired so its layout rules apply normally.

JavaScript syntax and the existing six Node test files pass; native desktop visual testing remains unverified here.

## Search toolbar and shortcut-window update

The action toolbar now has **Search** immediately after **Edit Client**. It opens Global Find, which already searches by client name and document name across the signed-in company's active workspace, as well as descriptions, folder locations and notes. The search field explicitly describes client/document search. The previous standalone Global Find toolbar group was replaced by this button; the Edit menu and keyboard shortcut still open the same search.

The Keyboard Shortcuts window no longer includes Add Binding. Select a numbered existing binding and use Modify or Clear; Restore Defaults and Save remain available.

JavaScript syntax and the six existing Node test files pass. Native desktop visual testing remains unverified here.

## Calendar Clear Status and numbered shortcut editor

Right-click a scheduled client or search result and choose **Clear Status** to remove Drafting, Waiting on reports, or Completed. Clearing Completed removes the green check without removing the client from the calendar. Client return types already appear on weekly client chips; they now also appear in full-year search results, using the current client profile's return type. Profiles without a type show Not set.

Under **Settings → Keyboard Shortcuts**, each binding has a numbered row. Select a row, click **Modify**, then choose Ctrl, Alt and/or Shift checkboxes and a key from the dropdown. Select **OK**, then **Save** in the mapper. Alternate bindings are modified independently; **Add Binding** adds an alternative and **Clear** removes just the selected binding. The dropdown supports A–Z, top-row digits, F1–F24, navigation/editing keys, punctuation and distinct numeric keypad keys. Conflicts and reserved keys are reported before changes are applied. Notes formatting, dialog controls and the settings chord remain context-specific fixed controls as described below.

JavaScript syntax and all six Node test files pass, with additional tests for numeric keypad distinction, shifted top-row digits, individual alternate edits and conflicting changes. Go tests now include clearing a status. Go/native desktop visual verification remains to be run locally.

## Expanded firm calendar, client search and workflow status

The Firm Calendar opens nearly full-window, with the weekly schedule and document pane using the available screen. Search clients by name across every assignment in the entire 2026 calendar, regardless of the displayed week. Results include dates; click a result to jump to its week and view client documents. Clear removes the search.

Right-click a scheduled client or search result to select **Drafting**, **Waiting on reports**, or **Completed**. Completed displays a green check mark; changing to another status removes it. Status belongs to the selected date/client assignment, persists in the company's encrypted SQLite calendar metadata, and follows that assignment when rescheduled. Other occurrences of the same client are independent. Existing assignments start without a status. Search finds scheduled clients, not unscheduled clients from the document tree.

JavaScript syntax and six Node test files pass, including year-wide search, case-insensitive matching, date sorting, and status badges. Go regression tests cover targeted updates, allowed statuses, legacy compatibility and JSON round trips; Go and native desktop visual tests cannot run in this environment and need local verification.

## Keyboard shortcut mapper

Open **Settings → Keyboard Shortcuts**. Click an action's shortcut field, press the desired keys, then choose Save. Assignments replace that action's existing shortcuts. Disable clears an action; Restore Defaults stages the original mappings and requires Save to apply. Cancel discards changes. Duplicate, unsupported, and reserved keys are rejected. Choices are saved per company/user in this desktop webview's local settings. Clearing those settings restores defaults; these choices are not part of the encrypted document backup.

Main shortcuts do not run while editing text or while dialogs are open, except Help, Sign out, and the context-appropriate PDF search. Notes formatting and save keys, dialog keys, Shift-click document selection, and G then S for Settings stay fixed. Ctrl+B remains Bold inside notes. Menu shortcut hints and the main Help shortcut list reflect saved assignments. Operating-system shortcuts may intercept a combination before BabyFileCab receives it.

JavaScript syntax and five Node test files pass, including mapper conflicts, reserved keys, disabled actions, persistence, malformed preferences, and company/user isolation. Native desktop visual testing remains to be performed locally.

## Preview and missing-key hardening

Formatted notes are rebuilt from allowed text/formatting nodes, with a strict attribute and style policy. Active elements and foreign namespaces are discarded; descendants of unsupported wrappers are still checked. Go sanitizes formatted notes on both save and read, including existing notes. HTML paste is sanitized before insertion, and native content drops into the editor are blocked. Image previews use a validated raster data URL assigned through the DOM, rather than interpolating it into HTML. Unsupported styles, embedded images, links, and active content are intentionally removed from notes. Formatted notes are limited to 2 MiB and 128 nesting levels.

Login NEVER generates a company key for an existing account. New company creation still creates a random key. Missing key files fail closed before migrations or session creation, even for administrators and empty/missing key directories. Restore the correct encrypted key file from a complete backup, or enroll an unversioned account using another administrator who already holds this company's key. An account with a missing versioned key must restore that key file. If every copy of the company key is lost, this update cannot recover the encrypted documents. Legacy installations that relied on first-login key creation now require a separately authorized migration/recovery process; this build deliberately does not guess whether an existing vault is safe to initialize.

Validation: `node --check frontend/dist/app.js` and all four Node test files pass. `preview_security_test.go` adds malicious notes and missing-key regression cases. Go is unavailable here, so run `go test -tags webkit2_41 ./...` locally. A Playwright browser test is included at `frontend/tests/notes-sanitizer.browser.cjs`; it was not run successfully because this environment has no installed browser. Parser and desktop integration remain to be verified locally. Back up your entire data directory before upgrading.

## Change Password

Use **Settings → Change Password**. Enter your current password and confirm a new password of at least 8 characters. The account verifier uses Argon2id with a fresh salt. The existing company key is rewrapped with a fresh salt and nonce; documents and other users are unchanged. A synced, separate encrypted key file is staged before the account record atomically switches to it. Never downgrade to an older build after changing passwords: older versions do not understand the new key-file reference. Back up the complete data directory, including hidden account and vault-key files, together. Old backups remain protected by their original passwords; changing the password does not revoke copied backups.

Validation: JavaScript checks run in this environment. Go tests were added for password validation, restart persistence, both account stores, unchanged document keys, other-user access, staged wrappers, and expired sessions. A Go toolchain is unavailable here; run `go test ./... -tags webkit2_41` locally before using the update with live data.

## Configurable auto-lock — September 29, 2026

Settings → Auto-lock after inactivity offers 15, 30, or 45 minutes. The default is 15 minutes. The preference is saved for this computer and applies to both the desktop input timer and backend session expiry. Sleep and OS-session locking remain separate triggers.

JavaScript syntax, all three timeout simulations, and document selection tests passed. Go tests for preference persistence, validation, and session expiry were added but could not run here because Go is unavailable.

## Multiple document selection — September 29, 2026

Click the first file, then hold Shift and left-click another file to select the visible range in that client. Collapsed folders and other clients are excluded. Click normally to select one file again. Delete confirms the count and deletes selected files individually; notes.txt remains protected. Dragging and renaming still operate on the active individual file.

Validation: `node frontend/tests/tree-selection.test.cjs` and JavaScript syntax checks passed. Native Wails interaction still needs a desktop test.

## Section and inactivity fixes — September 29, 2026

- Add Section creates an empty sibling when a section is selected. Existing files are not moved.
- Delete Section requires a completely empty directory, including no hidden files or subsections.
- Sessions expire after the selected 15, 30, or 45 minutes without input. Switching windows and minimizing no longer trigger a short logout. Linux sleep/session locking is scoped to the current process session.
- Frontend JavaScript syntax and simulated inactivity behavior passed. Go regression tests were added but could not run here because Go is unavailable. Native sleep/account-switch behavior still needs a Parrot runtime check.

Run from this directory:

```sh
gofmt -w section_session_test.go session_linux.go
go test ./...
wails dev -tags webkit2_41
```

# BabyFileCab — Wails User Manual & Keybindings

This build continues the local-first Go + Wails BabyFileCab codebase with company users, grouped actions, client communications search, return-type display, light/dark themes, and the Edit Client context-menu workflow.

## User Manual

A new **Help → User Manual** screen contains the complete keyboard-shortcut reference. Press **F1** anywhere in the signed-in BabyFileCab workspace to open it.

## Main application shortcuts

- `F1` — User Manual / Help & Support
- `Ctrl+Shift+F` — Global Find
- `Ctrl+F` — Find in the selected searchable PDF
- `M` — Find in PDF (same as Ctrl+F)
- `F2` — Rename selected document
- `Ctrl+B` — Back Up Client chooser
- `F5` — Refresh workspace
- `Ctrl+Shift+L` — Sign out
- `Ctrl+Q` — Exit BabyFileCab

## Fast workflow keys

These work in the main workspace while the user is not typing in an editable field and no secondary modal is open.

- `f` — Upload file(s)
- `Shift+F` — Rename selected document
- `c` — Add client
- `Shift+C` — Edit selected client
- `y` — Add tax year
- `Shift+Y` — Edit selected tax year
- `s` — Add section
- `Shift+S` — Edit selected section
- `d` or `D` — Delete the highlighted client, tax year, section, or document with normal confirmation/safety checks
- `m` — Manage Users
- `g` then `s` — Open Settings

## Client notes

When the rich `notes.txt` editor has focus:

- `Ctrl+S` — Save notes
- `Ctrl+B` — Bold
- `Ctrl+I` — Italic
- `Ctrl+U` — Underline

The notes shortcuts take priority over application shortcuts. For example, `Ctrl+B` means **Bold** while editing notes and **Back Up Client** elsewhere in the main workspace.

## Document Properties

- `Ctrl+S` — Save the document description/keywords
- `Esc` — Save and close Document Properties

## Search windows

- `Enter` — Run Global Find when its search box has focus
- `Enter` — Open a focused Global Find result
- `Enter` — Move to the next PDF-search match
- `Esc` — Clear/close Find in PDF

## Manage Users

Settings now includes **Manage Users…**. The `m` shortcut opens the same screen. It lists users belonging to the signed-in company without exposing password hashes or salts. Administrators can choose **Add User** or enroll an existing user whose vault access is pending. The administrator first unlocks the company vault; the existing user then enters their own current password in the enrollment dialog. The administrator should not ask the user to disclose it.

## Shared company data

Every account is assigned to a company. Users under the same company on the same BabyFileCab installation share that company's local client Work Space and documents.

Company workspaces are stored under:

`~/BabyFileCabData/companies/<company-id>/`

The account index is stored in `~/BabyFileCabData/.accounts.sqlite` after migration. Each company keeps sensitive profiles, descriptions, and calendar entries as encrypted records in `.vault.sqlite`. Password verifiers use Argon2id for new accounts; old PBKDF2 verifiers are upgraded after successful sign-in. Wrapped vault keys live separately in `~/BabyFileCabData/.vault-keys/`.

## Run on Parrot OS

Extract the ZIP under `~/Desktop/neovim`, then run:

```bash
cd ~/Desktop/neovim/BabyFileCab-Wails-CREATE-INVOICE
go mod tidy
wails dev -tags webkit2_41
```

For later runs:

```bash
cd ~/Desktop/neovim/BabyFileCab-Wails-CREATE-INVOICE
wails dev -tags webkit2_41
```

## Local-first limitation

This remains a local-first build. Separate computers do not automatically synchronize yet; that would require a future network/server/cloud synchronization layer.


## Latest workflow improvements

- Right-click a section to edit its name, upload files directly to it, or delete it.
- Sign Out is available under the File menu and is no longer duplicated in the top-right header.
- Edit -> Unarchive Clients opens a searchable list of archived clients and restores the selected client to the active Work Space.
- The Edit menu no longer contains Edit Selected Client. Client editing remains available from the Client toolbar, File menu, right-click client menu, and Shift+C.
- Help -> About BabyFileCab now explains the application's local-first accounting-document-management purpose and intended users.


## 2025 / 2026 Federal Tax Calculator

The top application menu now includes **Tools → Federal Tax Calculator…**. Inside the calculator, a **Tax year** dropdown lets the user choose **2025** or **2026**.

The 2025 calculation continues to use the updated 2025 IRS ordinary-income brackets, standard deductions, capital-gain thresholds, and 2025 Social Security wage base.

The 2026 calculation uses the 2026 federal bracket structure referenced by the Tax Foundation and the IRS Revenue Procedure 2025-32 values for exact filing-status thresholds, standard deductions, and capital-gain thresholds. The 2026 Social Security wage base is $184,500, as published by the Social Security Administration.

The calculator accepts filing status, wages, self-employment income, taxable interest, and short- and long-term capital gains/losses. It remains a planning estimate and does not replace tax-preparation software or a complete Form 1040 computation.

## Child Tax Credit update

The Federal Tax Calculator now estimates the Child Tax Credit (CTC) and the earned-income method of the Additional Child Tax Credit (ACTC) for both 2025 and 2026.

- Maximum CTC: $2,200 per qualifying child for 2025 and 2026.
- Maximum refundable ACTC: $1,700 per qualifying child for 2025 and 2026.
- Phaseout starts at $400,000 modified AGI for Married Filing Jointly and $200,000 for all other filing statuses.
- ACTC estimate uses 15% of earned income above $2,500, subject to the unused CTC and per-child refundable limit.

The calculator asks for the number of qualifying children under age 17 and assumes the taxpayer and children otherwise meet the IRS eligibility and Social Security number requirements. It does not implement the special alternative ACTC calculation that may apply to some taxpayers with three or more qualifying children, Form 2555 interactions, Puerto Rico rules, or other credit-limit interactions.


## Estimated quarterly payments

The 2025/2026 federal tax calculator now accepts expected federal income-tax withholding and shows a simple equal-quarter planning schedule. It displays IRS Form 1040-ES due dates for the selected year and includes a button that opens the official IRS payment page. The installment estimate is not a substitute for Form 1040-ES safe-harbor or annualized-income calculations.


## 2026 Firm Calendar

The **Tools** menu now includes **Firm Calendar…**. The Firm Calendar is a company-shared local weekly scheduling view for calendar year 2026.

- The left arrow moves back one week and the right arrow moves forward one week.
- The **Week of 2026** dropdown jumps directly to any weekly calendar view for the year.
- **Current Week** jumps to the current 2026 week when applicable.
- Click the **+** button on any 2026 date to assign a client. Multiple clients can be assigned to the same date.
- Click a scheduled client name to see the client's Permanent Folder and documents grouped by tax year.
- Right-click a scheduled client and choose **Reassign to New Date…** to move that client to another date in 2026. BabyFileCab jumps the weekly calendar to the destination week after the move.
- Click a document in the calendar's client-document panel to open that document in BabyFileCab's main Work Space and preview pane.
- Click **×** next to a scheduled client to remove only that calendar assignment; the client and documents are not deleted.

The schedule is stored inside the signed-in company's local BabyFileCab workspace as `.firm-calendar-2026.json`, so local users under the same company see the same Firm Calendar. The current local-first limitation still applies: separate computers do not synchronize automatically.


## Password visibility eye buttons

Password fields now include an eye icon that lets the user temporarily show or hide the password they typed. The toggle is available on Sign In, Create Company Account, Create User Under Existing Company, Confirm Password, and Administrator Authorization fields. Each password field toggles independently and is reset to hidden when authentication/account screens are reopened.


## Firm Calendar return type display

Scheduled clients in the 2026 Firm Calendar now show the client's tax return type directly beside the client name: **1040**, **1065**, **1120S**, or **1120**. The selected-client document panel and the right-click calendar menu also show the return type. The value comes from the client's existing BabyFileCab profile, so edits to a client's return type are reflected in the calendar.


## Work Space label and secure accounting subtitle

The main left pane is now labeled **Work Space** instead of Document Tree. Under the BabyFileCab name, the header now reads **Secure document work space for accounting and tax professionals**.

## Company-wide Audit Trail

The **Tools** menu now includes **Audit Trail…**. This is a company-wide history for the signed-in BabyFileCab workspace. It shows the date/time, user, affected client when available, action, and details. The window can be searched and refreshed.

Company-wide and per-client audit entries are encrypted records in the company `.vault.sqlite` database. Important client/document changes, backups/restores, archive/unarchive actions, Firm Calendar changes, user creation, sign-in/sign-out, app exit, and selected app/tool events are recorded. Existing `.audit.jsonl` and `.babyfilecab-audit.jsonl` history is imported during migration; older entries may not identify the user. Client backups preserve the company audit database in place when restored, but do not export its history into the client archive.

## File → Back Up Client Files

The File menu now places **Back Up Client Files…** directly below **Client Communications…**. Opening it displays the full active client list. Choose one client and BabyFileCab will ask where to save an encrypted `.bfcbackup` archive. Restoring older plaintext `.zip` archives remains available; the old ZIP itself remains unencrypted until you remove or protect it. `Ctrl+B` opens the same backup workflow.



## Tools → Engagement Letter

The Wails build now includes **Tools → Engagement Letter…**, based on the automated Engagement Letter workflow from the earlier Python BabyFileCab version.

- Select an active BabyFileCab client; client name, address, phone, email, client ID, and return type come from the local client profile.
- Select a preparer from the company&apos;s Administrator/Staff users; the letter uses that user&apos;s stored representative address, phone, and email.
- Services: **1040, 1041, 1065, 1120S, 1120, Bookkeeping, Tax Projection**. The Service dropdown is independent of the client profile, so the user explicitly chooses the engagement being generated.
- Enter the tax year/service period, choose **Flat Fee** or **Hourly Rate**, enter the fee/rate, and choose the letter date.
- A live scrollable preview updates as the form changes.
- Generate **PDF** or **Microsoft Word (.docx)** from the buttons at the bottom of the generator. The generated file opens automatically after creation.
- By default, files are stored under the company workspace in `Generated Forms/Engagement Letters`; the user can browse to a different output location.
- The letter retains the Python version&apos;s detailed scope, client responsibilities, reliance/limitation, tax-position/e-file language for return engagements, fee/out-of-scope terms, records/confidentiality, termination, and client/firm signature blocks.
- Creating a letter records both a per-client audit event and a company-wide BabyFileCab Audit Trail event.

The generated files are ordinary PDF/Word files and are not automatically copied into the client&apos;s document folders. The engagement-letter wording remains a general template that should be reviewed by the firm&apos;s attorney, professional-liability insurer, or another qualified adviser before production use.

## Automated Engagement Letter — client/preparer autofill

The Tools > Engagement Letter form can select any active BabyFileCab client and automatically loads the saved client name, address, email, phone, client ID, and return type. Selecting a company user as preparer automatically loads the firm name plus the preparer's saved address, email, and phone into the form and generated letter.

Engagement types include Bookkeeping, Individual Return (1040), Partnership Return (1065), S Corporation Return (1120-S), C Corporation Return (1120), Estate/Trust Return (1041), and Tax Projection. The fee arrangement supports either a flat fee or hourly rate. The completed engagement letter can be generated as PDF, Microsoft Word (.docx), or both.


## Engagement Letter — Python-style automated workflow

The Wails Engagement Letter now mirrors the earlier BabyFileCab Python workflow: choose an existing client, choose a preparer, select the service and fee arrangement, review the live letter preview, and generate PDF, Microsoft Word, or both. Client address/phone/email and preparer/firm contact information are pulled from the profiles already stored in BabyFileCab rather than re-entered into the form.

## CPA Firm Engagement Letter Generator

The Tools menu now contains **Engagement Letter Generator…** with the requested CPA-firm workflow.

- The current BabyFileCab **Company Account** name is displayed at the top.
- **Browse Logo…** accepts PNG or JPEG images, saves the selected firm logo for that company, and uses it in the preview plus generated PDF/Word letters.
- **Client** is selected from the active BabyFileCab client database; saved client address, phone, email, client ID, and return type remain available to the letter workflow.
- **Preparer** is selected from the company’s Administrator/Staff users; the selected preparer’s saved address, phone, and email are printed in the firm letterhead.
- **Service** choices: 1040, 1041, 1065, 1120S, 1120, Bookkeeping, and Tax Projection.
- **Tax Year** choices: 2018 through 2026.
- **Fee Arrangement** choices: Flat Fee or Hourly Rate, with an amount entered in U.S. dollars.
- The live **Letter Preview** shows the firm logo, company name, preparer/contact information, client information, Engagement Letter title, service, tax year, fee arrangement, and the engagement terms.
- Generate PDF or Microsoft Word (.docx) from the two buttons at the bottom of the generator.

The firm logo is stored locally inside the signed-in company workspace as `.engagement-letter-logo.jpg`. It is company-specific and remains local-first with the rest of BabyFileCab.

## Final Engagement Letter generator workflow

The **Tools → Engagement Letter Generator…** window is a complete CPA-firm engagement-letter generator. After choosing the company logo, BabyFileCab client, preparer, service, tax year (2018–2026), fee arrangement, and fee/rate, the live preview shows the actual engagement letter that will be created.

At the very bottom are two direct output buttons:

- **Generate PDF Engagement Letter** — opens a Save As dialog for a `.pdf`, creates the PDF, and opens the finished file.
- **Generate MS Word Engagement Letter** — opens a Save As dialog for a `.docx`, creates the Microsoft Word document, and opens the finished file.

There is no separate output-path form and no extra “Generate Both” step. The user completes the engagement information, reviews the letter, then chooses the final document format from the bottom of the generator.


## Rebuilt Engagement Letter Generator

This build replaces the previous Engagement Letter screen with a clean form built around native dropdown controls. Every time the tool opens it reloads the active BabyFileCab client database, then presents explicit dropdowns for **Client**, **Preparer**, **Service**, **Tax Year**, and **Fee Arrangement**. Service selection is no longer changed automatically from the client return type. The current signed-in Company Account appears at the top, the company-specific firm logo can be browsed/removed, and the letter preview is directly below the form. The bottom buttons are **Generate PDF** and **Generate MS Word**.

- Engagement Letter Generator: Generate MS Word and Generate PDF buttons are positioned at the bottom of the generator, directly after the letter preview.

## Engagement Letter Generator — scroll and document buttons

The Engagement Letter Generator window now has an explicit vertical scrollbar. Scroll to the bottom of the letter preview to reach **Generate MS Word Document** and **Generate PDF Document**. Both buttons remain wired to the existing DOCX/PDF generation workflow.


## Create an Invoice

The **Tools** menu now includes **Create an Invoice…**. The invoice generator follows the same desktop-form pattern as the Engagement Letter Generator:

- Company Account automatically comes from the company signed into BabyFileCab.
- Firm Logo uses the company logo already stored for generated firm documents, with Browse Logo and Remove Logo controls.
- Client is a real dropdown populated from active BabyFileCab clients. Selecting a client loads the saved name, address, phone, and email.
- Preparer is a real dropdown populated from company users.
- Service is a dropdown for 1040, 1041, 1065, 1120S, 1120, Bookkeeping, and Tax Projection.
- Tax Year runs from 2018 through 2026.
- Invoice Number, Invoice Date, and Due Date are editable.
- Fee Arrangement can be Flat Fee or Hourly Rate. Hourly invoices show an Hours field and automatically calculate hours × rate.
- Optional Description and Invoice Note fields appear on the finished invoice.
- A live invoice preview is shown before generation.
- The generator is vertically scrollable, with **Generate MS Word Document** and **Generate PDF Document** buttons at the bottom.
- Generated invoices are logged to both the client history and the company-wide Audit Trail.

The default generated-forms location is under `Generated Forms/Invoices`, although the user can choose the final file location in the Save As dialog.

## Invoice generator scrolling
The Create an Invoice window includes an always-visible vertical scrollbar so the entire invoice form, preview, and the Generate MS Word Document / Generate PDF Document buttons can be reached by scrolling up and down.

## Create an Invoice — visible in-window scrollbar
The Create an Invoice popup now includes a dedicated, always-visible scrollbar on the right side of the invoice window. The up/down arrow buttons, scrollbar track, draggable thumb, mouse wheel, and keyboard scrolling all control the invoice form/preview area so the user can reach the Generate MS Word Document and Generate PDF Document buttons at the bottom.


## Create an Invoice scrollbar
The Create an Invoice window uses one vertical scrollbar for the complete invoice form, preview, and Generate MS Word / Generate PDF buttons. The extra custom scrollbar and nested preview scrollbar were removed.


## Currency Converter (IRS)
Tools → Currency Converter (IRS)… converts between U.S. dollars and currencies in the IRS yearly-average exchange-rate table for 2021–2025, includes both conversion directions, displays the exact rate/formula used, and links to the IRS source page.


## Vault migration and limits

On the first administrator sign-in, BabyFileCab makes a random 256-bit key for that company, wraps it with a key derived from the administrator password using Argon2id, and migrates that company’s existing files to AES-256-GCM. Each file is replaced atomically after an encrypted backup and a decryption check. A failed or interrupted migration resumes on the next sign-in; the vault is not marked ready until it finishes. Keep your own separate backup before upgrading. Existing users show **Pending** in Manage Users and cannot sign in to the vault until the administrator enrolls them with their own password.

Client fields, document descriptions, calendar data, and user profiles are encrypted in SQLite. The login index and the company name needed on the account screen are readable locally. Client and document **names and folder structure** are still visible on the local filesystem. The program does not yet sync to Appwrite. Encrypted backup/restore currently has a 512 MB per-client limit.

PDF rendering and external desktop applications require short-lived decrypted working copies. BabyFileCab puts them in a private preview directory, removes them on sign-out/lock, and removes leftovers on the next startup after a crash. An exported PDF or Word file deliberately saved outside the vault is plaintext. Full-disk encryption remains valuable for temporary-file and deleted-file remnants. Do not upload raw `BabyFileCabData` folders to a cloud service without reviewing these metadata and key-management limits.

Linux listens for login1 sleep, lock, and user-switch events. On other systems the window visibility/focus handlers and a wake-time gap check trigger locking; native Windows and macOS session notifications have not been implemented. Verify lock behavior on the target OS before using client data there.

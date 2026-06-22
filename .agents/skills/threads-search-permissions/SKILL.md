---
name: threads-search-permissions
description: Guidelines on Meta App Review, Live/Development modes, and resolving keyword search permissions on the Threads Graph API.
---

# Threads Search & Permissions Guide

This guide covers the troubleshooting procedures, development workflows, and App Review prerequisites for Meta's Threads Graph API keyword search functionality.

## Troubleshooting permission errors

When calling the `GET /keyword_search` endpoint, if you encounter a `THApiException (code 10)` with the message `"Application does not have permission for this action"`, this indicates a configuration mismatch between your app's requested scopes, its current approval status, and the active token's permissions.

### Understanding Development Mode vs. Live Mode Gating

* **Development Mode (Testing):**
  * Newly created apps start in Development Mode for testing.
  * Access to advanced features (like public keyword search) is strictly limited to users with roles on the app (Administrators, Developers, and Testers).
  * Global search is blocked in this mode. You can generally only query or pull data from accounts explicitly added as **Threads Testers** under your App Dashboard roles.
* **Live Mode & App Review:**
  * Simply switching the dashboard to **Live Mode** will not authorize public keyword search if the permission has not passed **App Review**.
  * Switching to Live Mode without App Review approval actually strips unapproved capabilities away from anyone who isn't an admin or developer of the app.

---

## Prerequisite Workflow for Production / Public Search

To successfully search public Threads content in a production environment:

1. **Keep the app in Development Mode** while building and testing your implementation.
2. Complete **Business Verification** inside your Meta Developer Account (usually required for advanced data search endpoints).
3. Submit the `threads_keyword_search` permission for **App Review**. You will need:
   * A privacy policy URL.
   * Terms of service.
   * A recorded screencast showing exactly how your application fetches, processes, and displays the public search data.
4. **Only after Meta approves the permission** should you switch your app to **Live Mode**.

---

## Solo Builder Shortcut (Bypassing App Review)

If the application is a personal project or internal tool querying data purely on behalf of your own account:

* **Do not switch to Live Mode.**
* Keep the app in **Development Mode** indefinitely.
* Add your personal Threads account as a **Threads Tester** under the Roles tab in the Meta Developer Dashboard.
* Authenticate the tester account to generate your access token. This bypasses the App Review requirement entirely.

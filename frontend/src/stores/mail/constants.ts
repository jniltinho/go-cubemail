/**
 * @file constants.ts
 * @description Static mapping constants for the Cubemail mail store.
 */

/**
 * Maps folder names received from the IMAP backend server (localized or system defaults)
 * to simplified routing identifiers utilized on the frontend client.
 */
export const FOLDER_ID_MAP: Record<string, string> = {
  'Inbox': 'inbox', 'Starred': 'starred', 'Sent Items': 'sent', 'Sent': 'sent',
  'Drafts': 'drafts', 'Archive': 'archive', 'Junk Mail': 'junk', 'Junk': 'junk',
  'Deleted Items': 'trash', 'Trash': 'trash', 'INBOX': 'inbox',
}

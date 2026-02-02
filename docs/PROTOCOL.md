# Z39.50 Protocol Implementation Details

This document outlines the technical specifics of the Z39.50 (ISO 23950) protocol implementation within the **Open-Z3950-Gateway**.

## Overview

The gateway implements a persistent, stateful Z39.50 client and server stack written in Go. It uses direct TCP connections and handles BER (Basic Encoding Rules) encoding/decoding manually to ensure maximum compatibility with legacy library systems.

## Supported Facilities

The implementation supports the following Z39.50 services (PDUs):

| PDU | Tag (Hex) | Description | Support Level |
| :--- | :--- | :--- | :--- |
| **Initialize** | `20` / `21` | Session establishment and capability negotiation. | Full (v3) |
| **Search** | `22` / `23` | Query submission using Type-1 (RPN) queries. | Full (Recursive) |
| **Present** | `24` / `25` | Retrieval of records from a result set. | Full |
| **Scan** | `35` / `36` | Browsing term indexes (e.g., list authors near "Smith"). | Partial (Term/Count) |
| **Delete** | `30` / `31` | Deleting result sets to free server resources. | Basic (Delete All) |
| **Close** | `48` | Graceful session termination. | Full |

## Initialization Parameters

When connecting to remote targets, the client proposes:

*   **Protocol Version**: Z39.50 v3 (`0x20` / bit 2 set).
*   **Options**: Search (`0x80`) and Present (`0x40`) = `0xC0`.
*   **Message Size**:
    *   Preferred Message Size: **65,536 bytes** (64KB)
    *   Maximum Record Size: **65,536 bytes** (64KB)

## Query & Search Support

The gateway implements a fully recursive **Type-1 (RPN)** query engine.

### Boolean Logic
It supports arbitrarily complex query trees constructed using:
*   **AND** (`opVal=0`)
*   **OR** (`opVal=1`)
*   **AND-NOT** (`opVal=2`)

### Attribute Set
The implementation uses the **Bib-1** attribute set (`1.2.840.10003.3.1`).

| Use Attribute | ID | Name |
| :--- | :--- | :--- |
| **Personal Name** | `1` | Author (Personal) |
| **Corporate Name** | `2` | Author (Corporate) |
| **Title** | `4` | Title |
| **Title Series** | `5` | Series Title |
| **ISBN** | `7` | International Standard Book Number |
| **ISSN** | `8` | International Standard Serial Number |
| **Subject** | `21` | Subject Heading |
| **Date** | `31` | Date of Publication |
| **Author (Gen)** | `1003` | Generic Author |
| **Any** | `1016` | Keyword (Any Field) |

## Record Syntax & Encoding

The client requests records using specific Object Identifiers (OIDs) in the `PresentRequest`.

### Supported OIDs
*   **MARC 21**: `1.2.840.10003.5.10` (Default)
*   **UNIMARC**: `1.2.840.10003.5.1`
*   **SUTRS**: `1.2.840.10003.5.101` (Simple Unstructured Text)

### Character Encoding Strategy
Library systems use a variety of legacy character encodings. The gateway's `DecodeText` function implements a heuristic strategy:

1.  **UTF-8 Validation**: Checks if the data is valid UTF-8.
2.  **CJK Fallback**: Tries to decode as **GBK**, **Big5**, **ShiftJIS**, **EUC-JP**, or **EUC-KR**.
3.  **Auto-Detection**: Uses `golang.org/x/net/html/charset` to detect other legacy encodings (e.g., Latin1).
4.  **Raw Fallback**: Returns raw bytes string if all else fails.

## Architecture Notes

*   **Connection Pooling**: The gateway manages a pool of persistent TCP connections to remote targets to avoid the overhead of re-handshaking for every user request.
*   **Stateless Frontend**: The React frontend is stateless; the Go backend maintains the Z39.50 session state (Result Sets) mapped to user sessions.

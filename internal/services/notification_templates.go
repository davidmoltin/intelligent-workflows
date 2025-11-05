package services

// Email template for new approval request
const approvalRequestEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
        .content { background-color: #f9f9f9; padding: 30px; border: 1px solid #ddd; border-top: none; }
        .info-row { margin: 10px 0; padding: 10px; background-color: white; border-left: 3px solid #4CAF50; }
        .label { font-weight: bold; color: #555; }
        .value { color: #333; }
        .button { display: inline-block; padding: 12px 30px; margin: 20px 10px 10px 0; background-color: #4CAF50; color: white; text-decoration: none; border-radius: 5px; }
        .button.reject { background-color: #f44336; }
        .footer { text-align: center; padding: 20px; color: #777; font-size: 12px; }
        .warning { background-color: #fff3cd; border-left: 3px solid #ffc107; padding: 10px; margin: 15px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔔 Approval Request</h1>
        </div>
        <div class="content">
            <p>A new approval request requires your attention.</p>

            <div class="info-row">
                <span class="label">Request ID:</span>
                <span class="value">{{.RequestID}}</span>
            </div>

            <div class="info-row">
                <span class="label">Entity Type:</span>
                <span class="value">{{.EntityType}}</span>
            </div>

            <div class="info-row">
                <span class="label">Entity ID:</span>
                <span class="value">{{.EntityID}}</span>
            </div>

            <div class="info-row">
                <span class="label">Approver Role:</span>
                <span class="value">{{.ApproverRole}}</span>
            </div>

            {{if .Reason}}
            <div class="info-row">
                <span class="label">Reason:</span>
                <span class="value">{{.Reason}}</span>
            </div>
            {{end}}

            {{if .ExpiresAt}}
            <div class="warning">
                ⏰ <strong>Expires:</strong> {{.ExpiresAt}}
            </div>
            {{end}}

            <div style="margin-top: 30px; text-align: center;">
                <a href="{{.ApprovalURL}}/approve" class="button">✅ Approve</a>
                <a href="{{.ApprovalURL}}/reject" class="button reject">❌ Reject</a>
            </div>

            <p style="margin-top: 20px; font-size: 14px; color: #666;">
                Or view details at: <a href="{{.ApprovalURL}}">{{.ApprovalURL}}</a>
            </p>
        </div>
        <div class="footer">
            <p>Intelligent Workflows - Approval System</p>
            <p>Requested at: {{.Timestamp}}</p>
        </div>
    </div>
</body>
</html>
`

// Email template for approved approval
const approvalApprovedEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
        .content { background-color: #f9f9f9; padding: 30px; border: 1px solid #ddd; border-top: none; }
        .info-row { margin: 10px 0; padding: 10px; background-color: white; border-left: 3px solid #4CAF50; }
        .label { font-weight: bold; color: #555; }
        .value { color: #333; }
        .success { background-color: #d4edda; border-left: 3px solid #28a745; padding: 15px; margin: 20px 0; border-radius: 5px; }
        .footer { text-align: center; padding: 20px; color: #777; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>✅ Approval Approved</h1>
        </div>
        <div class="content">
            <div class="success">
                <strong>Great news!</strong> Your approval request has been approved.
            </div>

            <div class="info-row">
                <span class="label">Request ID:</span>
                <span class="value">{{.RequestID}}</span>
            </div>

            <div class="info-row">
                <span class="label">Entity Type:</span>
                <span class="value">{{.EntityType}}</span>
            </div>

            <div class="info-row">
                <span class="label">Entity ID:</span>
                <span class="value">{{.EntityID}}</span>
            </div>

            {{if .Reason}}
            <div class="info-row">
                <span class="label">Original Reason:</span>
                <span class="value">{{.Reason}}</span>
            </div>
            {{end}}

            <p style="margin-top: 20px;">
                The workflow will continue execution automatically.
            </p>
        </div>
        <div class="footer">
            <p>Intelligent Workflows - Approval System</p>
            <p>Timestamp: {{.Timestamp}}</p>
        </div>
    </div>
</body>
</html>
`

// Email template for rejected approval
const approvalRejectedEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #f44336; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
        .content { background-color: #f9f9f9; padding: 30px; border: 1px solid #ddd; border-top: none; }
        .info-row { margin: 10px 0; padding: 10px; background-color: white; border-left: 3px solid #f44336; }
        .label { font-weight: bold; color: #555; }
        .value { color: #333; }
        .error { background-color: #f8d7da; border-left: 3px solid #dc3545; padding: 15px; margin: 20px 0; border-radius: 5px; }
        .footer { text-align: center; padding: 20px; color: #777; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>❌ Approval Rejected</h1>
        </div>
        <div class="content">
            <div class="error">
                Your approval request has been rejected.
            </div>

            <div class="info-row">
                <span class="label">Request ID:</span>
                <span class="value">{{.RequestID}}</span>
            </div>

            <div class="info-row">
                <span class="label">Entity Type:</span>
                <span class="value">{{.EntityType}}</span>
            </div>

            <div class="info-row">
                <span class="label">Entity ID:</span>
                <span class="value">{{.EntityID}}</span>
            </div>

            {{if .Reason}}
            <div class="info-row">
                <span class="label">Original Reason:</span>
                <span class="value">{{.Reason}}</span>
            </div>
            {{end}}

            <p style="margin-top: 20px;">
                The workflow execution has been halted.
            </p>
        </div>
        <div class="footer">
            <p>Intelligent Workflows - Approval System</p>
            <p>Timestamp: {{.Timestamp}}</p>
        </div>
    </div>
</body>
</html>
`

// Email template for expired approval
const approvalExpiredEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #9E9E9E; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
        .content { background-color: #f9f9f9; padding: 30px; border: 1px solid #ddd; border-top: none; }
        .info-row { margin: 10px 0; padding: 10px; background-color: white; border-left: 3px solid #9E9E9E; }
        .label { font-weight: bold; color: #555; }
        .value { color: #333; }
        .warning { background-color: #fff3cd; border-left: 3px solid #ffc107; padding: 15px; margin: 20px 0; border-radius: 5px; }
        .footer { text-align: center; padding: 20px; color: #777; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>⏰ Approval Expired</h1>
        </div>
        <div class="content">
            <div class="warning">
                Your approval request has expired without a decision.
            </div>

            <div class="info-row">
                <span class="label">Request ID:</span>
                <span class="value">{{.RequestID}}</span>
            </div>

            <div class="info-row">
                <span class="label">Entity Type:</span>
                <span class="value">{{.EntityType}}</span>
            </div>

            <div class="info-row">
                <span class="label">Entity ID:</span>
                <span class="value">{{.EntityID}}</span>
            </div>

            {{if .Reason}}
            <div class="info-row">
                <span class="label">Original Reason:</span>
                <span class="value">{{.Reason}}</span>
            </div>
            {{end}}

            {{if .ExpiresAt}}
            <div class="info-row">
                <span class="label">Expired At:</span>
                <span class="value">{{.ExpiresAt}}</span>
            </div>
            {{end}}

            <p style="margin-top: 20px;">
                The approval request has been automatically expired. If this was still needed, please create a new approval request.
            </p>
        </div>
        <div class="footer">
            <p>Intelligent Workflows - Approval System</p>
            <p>Timestamp: {{.Timestamp}}</p>
        </div>
    </div>
</body>
</html>
`

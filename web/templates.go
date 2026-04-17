package web

// Templates

const errorTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>Error - NahCloud</title>
    <link rel="icon" type="image/png" href="/static/logo.png">
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
    <style>{{.CSS}}</style>
</head>
<body class="bg-slate-50 min-h-screen flex items-center justify-center">
    <div class="text-center">
        <h1 class="text-2xl font-semibold text-slate-800 mb-2">Error</h1>
        <p class="text-slate-600 mb-4">{{.Message}}</p>
         <a href="/" class="text-[#2878B5] hover:underline">Go back</a>
    </div>
</body>
</html>`

const baseTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>NahCloud</title>
    <link rel="icon" type="image/png" href="/static/logo.png">
    <script src="https://unpkg.com/htmx.org@1.9.6"></script>
    <script>
        document.addEventListener('DOMContentLoaded', function() {
            document.body.addEventListener('htmx:beforeSwap', function(evt) {
                if (evt.detail.xhr.status === 400 || evt.detail.xhr.status === 500) {
                    evt.detail.shouldSwap = true;
                    evt.detail.isError = false;
                }
            });
        });
    </script>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400;500&family=IBM+Plex+Sans:wght@400;500;600;700&display=swap" rel="stylesheet">
    <style>{{.CSS}}</style>
    <style>
    :root {
        --surface: rgba(255, 255, 255, 0.94);
        --surface-strong: #ffffff;
        --line: #d8e3ee;
        --ink: #0f172a;
        --muted: #64748b;
        --accent: #2f6db5;
    }

    body {
        font-family: "IBM Plex Sans", sans-serif;
        background:
            radial-gradient(circle at top center, rgba(47, 109, 181, 0.12), transparent 34%),
            linear-gradient(180deg, #edf3f8 0%, #f7fafc 100%);
        color: var(--ink);
    }

    code, pre, .font-mono {
        font-family: "IBM Plex Mono", monospace;
    }

    .console-card {
        background: var(--surface);
        border: 1px solid var(--line);
        box-shadow: 0 20px 48px rgba(15, 23, 42, 0.08);
    }

    .console-card-strong {
        background: var(--surface-strong);
        border: 1px solid var(--line);
        box-shadow: 0 14px 30px rgba(15, 23, 42, 0.06);
    }

    .sidebar-link {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        border-radius: 0.75rem;
        padding: 0.75rem 1rem;
        color: #64748b;
        font-size: 0.875rem;
        font-weight: 500;
        transition: 150ms ease;
    }

    .sidebar-link:hover {
        background: #f8fafc;
        color: #0f172a;
    }

    .sidebar-link.active {
        background: #eff6ff;
        color: #0f172a;
    }

    .sidebar-link svg {
        flex-shrink: 0;
        opacity: 0.7;
    }
    </style>
</head>
<body class="min-h-screen">
    <div class="min-h-screen lg:flex">
        <aside class="w-full border-b border-slate-200 bg-white lg:sticky lg:top-0 lg:h-screen lg:w-60 lg:shrink-0 lg:overflow-y-auto lg:border-b-0 lg:border-r">
            <div class="border-b border-slate-200 px-6 pb-6 pt-6">
                <div class="flex items-center gap-3">
                    <img src="/static/logo.png" alt="NahCloud" class="h-10 w-10 rounded-lg">
                    <div>
                        <h1 class="text-xl font-bold text-[var(--accent)]">NahCloud</h1>
                        <p class="text-xs font-medium text-slate-500">Cloud Console</p>
                    </div>
                </div>
                {{if .Context.Org}}
                <div class="mt-4 rounded-xl bg-slate-50 px-4 py-3">
                    <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-slate-500">Current Org</p>
                    <p class="mt-2 text-sm font-semibold text-slate-900">{{.Context.Org.Name}}</p>
                    <p class="mt-1 text-xs text-slate-500">{{.Context.Org.Slug}}</p>
                </div>
                {{end}}
            </div>

            {{if .Context.Projects}}
            <div class="px-6 pb-4 pt-4">
                <label class="mb-1.5 block text-xs font-medium text-slate-500">Project</label>
                <select onchange="window.location.href='/projects/' + this.value + '?next={{if .Context.Section}}{{.Context.Section}}{{else}}instances{{end}}'" class="w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm transition-all focus:border-[#2878B5] focus:outline-none focus:ring-2 focus:ring-[#2878B5]/10">
                    {{range .Context.Projects}}
                    <option value="{{.Slug}}" {{if and $.Context.Project (eq .Slug $.Context.Project.Slug)}}selected{{end}}>{{.Name}}</option>
                    {{end}}
                </select>
            </div>
            {{end}}

            <nav class="px-3 pb-6">
                {{if .Context.Project}}
                <a href="/instances" class="sidebar-link {{if eq .Context.Section "instances"}}active{{end}}">
                    <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"></path>
                    </svg>
                    Instances
                </a>
                <a href="/storage" class="sidebar-link {{if eq .Context.Section "storage"}}active{{end}}">
                    <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"></path>
                    </svg>
                    Storage
                </a>
                {{end}}
                <a href="/projects" class="sidebar-link {{if eq .Context.Section "projects"}}active{{end}}">
                    <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"></path>
                    </svg>
                    Projects
                </a>
                <a href="/metadata" class="sidebar-link {{if eq .Context.Section "metadata"}}active{{end}}">
                    <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"></path>
                    </svg>
                    Metadata
                </a>
                <a href="/settings" class="sidebar-link {{if eq .Context.Section "settings"}}active{{end}}">
                    <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317a1 1 0 011.35-.936l1.488.595a1 1 0 00.873 0l1.488-.595a1 1 0 011.35.936l.116 1.598a1 1 0 00.51.816l1.373.786a1 1 0 01.39 1.39l-.798 1.369a1 1 0 000 .99l.798 1.369a1 1 0 01-.39 1.39l-1.373.786a1 1 0 00-.51.816l-.116 1.598a1 1 0 01-1.35.936l-1.488-.595a1 1 0 00-.873 0l-1.488.595a1 1 0 01-1.35-.936l-.116-1.598a1 1 0 00-.51-.816l-1.373-.786a1 1 0 01-.39-1.39l.798-1.369a1 1 0 000-.99l-.798-1.369a1 1 0 01.39-1.39l1.373-.786a1 1 0 00.51-.816l.116-1.598z"></path>
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path>
                    </svg>
                    Settings
                </a>
            </nav>
        </aside>

        <main class="min-w-0 flex-1">
            <div class="mx-auto min-h-screen max-w-6xl px-5 py-6 sm:px-6 lg:px-8 lg:py-8">
                <div id="content">
                    {{block "content" .}}{{end}}
                </div>
            </div>
        </main>
    </div>

    <div id="modal" class="fixed inset-0 z-50 hidden items-start justify-center bg-slate-950/60 px-4 pt-16 backdrop-blur-sm" onclick="if(event.target === this) this.style.display='none'">
        <div class="w-full max-w-lg overflow-hidden rounded-[24px] border border-slate-200 bg-white shadow-2xl" onclick="event.stopPropagation()">
            <div id="modal-content"></div>
        </div>
    </div>

    <style>
    #modal[style*="block"] { display: flex !important; }
    </style>

    <script>
        document.body.addEventListener('htmx:afterSwap', function(e) {
            if (e.target.id === 'modal-content') {
                document.getElementById('modal').style.display = 'block';
            }
        });
    </script>
</body>
</html>`

const settingsTemplate = `{{define "content"}}
<div class="max-w-5xl space-y-6">
    <div>
        <p class="text-xs font-semibold uppercase tracking-[0.22em] text-slate-500">Settings</p>
        <h2 class="mt-2 text-4xl font-semibold tracking-tight text-slate-900">Session And Sharing</h2>
        <p class="mt-3 max-w-2xl text-sm leading-6 text-slate-600">Manage this org, share a stable link, or reset everything back to an empty state.</p>
    </div>

    {{if .ResetDone}}
    <div class="rounded-3xl border border-emerald-200 bg-emerald-50 px-5 py-4 text-sm text-emerald-700 shadow-sm">
        Organization reset complete. This org is empty again and keeps the same permalink.
    </div>
    {{end}}

    <div class="grid gap-6 lg:grid-cols-[minmax(0,1.3fr)_minmax(0,0.9fr)]">
        <div class="space-y-6">
            <div class="rounded-[28px] border border-black/5 bg-white/85 p-6 shadow-[0_18px_40px_rgba(15,23,42,0.06)] backdrop-blur">
                <h3 class="text-lg font-semibold">Organization</h3>
                <div class="mt-5 grid gap-4 sm:grid-cols-2">
                    <div>
                        <p class="text-[11px] font-semibold uppercase tracking-[0.22em] text-slate-500">Name</p>
                        <p class="mt-2 text-sm font-medium text-slate-900">{{.Context.Org.Name}}</p>
                    </div>
                    <div>
                        <p class="text-[11px] font-semibold uppercase tracking-[0.22em] text-slate-500">Slug</p>
                        <p class="mt-2 font-mono text-sm text-slate-700">{{.Context.Org.Slug}}</p>
                    </div>
                </div>
            </div>

            <div class="rounded-[28px] border border-black/5 bg-white/85 p-6 shadow-[0_18px_40px_rgba(15,23,42,0.06)] backdrop-blur">
                <h3 class="text-lg font-semibold">Permalink</h3>
                <p class="mt-2 text-sm text-slate-600">Anyone with this link will enter the same org and get their own browser session for it.</p>
                <div class="mt-4 flex flex-col gap-3 sm:flex-row">
                    <input id="org-permalink" type="text" readonly value="{{.Permalink}}" class="w-full rounded-2xl border border-black/5 bg-slate-50 px-4 py-3 text-sm text-slate-700 focus:outline-none">
                    <button type="button" class="btn btn-secondary" onclick="copyPermalink()">Copy Link</button>
                </div>
            </div>
        </div>

        <div class="overflow-hidden rounded-[28px] border border-red-200 bg-white/85 shadow-[0_18px_40px_rgba(15,23,42,0.06)] backdrop-blur h-fit">
            <div class="border-b border-red-100 bg-red-50/70 px-6 py-5">
                <h3 class="text-lg font-semibold text-slate-900">Reset Organization</h3>
                <p class="mt-1 text-sm text-slate-600">Delete everything and return to the initial blank state.</p>
            </div>
            <div class="p-6">
                <p class="text-sm leading-6 text-slate-600">Reset removes all projects, instances, storage buckets, stored objects, and metadata. The org itself, its slug, and this permalink stay the same.</p>
                <form method="POST" action="/settings/reset" class="mt-6" onsubmit="return confirm('Reset this organization? This deletes all projects, instances, storage, and metadata.');">
                    <button type="submit" class="btn btn-danger">Reset Org</button>
                </form>
            </div>
        </div>
    </div>
</div>

<script>
function copyPermalink() {
    var input = document.getElementById('org-permalink');
    if (!input) return;
    input.focus();
    input.select();
    if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(input.value);
        return;
    }
    document.execCommand('copy');
}
</script>
{{end}}`

const projectsTemplate = `{{define "content"}}
<div class="overflow-hidden rounded-[28px] border border-black/5 bg-white/85 shadow-[0_18px_40px_rgba(15,23,42,0.06)] backdrop-blur">
    <div class="px-6 py-5 border-b border-slate-200 flex justify-between items-center">
        <h2 class="text-lg font-semibold">Projects</h2>
        <button class="btn btn-primary" hx-get="/projects/new?redirect_to=/projects" hx-target="#modal-content">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path>
            </svg>
            New Project
        </button>
    </div>
    <table class="w-full">
        <thead>
            <tr>
                <th class="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wider text-slate-500 bg-slate-50 border-b border-slate-200">Slug</th>
                <th class="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wider text-slate-500 bg-slate-50 border-b border-slate-200">Name</th>
                <th class="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wider text-slate-500 bg-slate-50 border-b border-slate-200">Created At</th>
                <th class="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wider text-slate-500 bg-slate-50 border-b border-slate-200">Actions</th>
            </tr>
        </thead>
        <tbody>
            {{range .Projects}}
            <tr class="hover:bg-slate-50">
                <td class="px-6 py-4 border-b border-slate-100">
                    <a href="/projects/{{.Slug}}?next=instances" class="font-medium text-[#2878B5] hover:underline">{{.Slug}}</a>
                </td>
                <td class="px-6 py-4 border-b border-slate-100">{{.Name}}</td>
                <td class="px-6 py-4 border-b border-slate-100 text-slate-500">{{.CreatedAt.Format "2006-01-02 15:04:05"}}</td>
                <td class="px-6 py-4 border-b border-slate-100">
                    <div class="flex gap-2">
                        <button class="btn btn-secondary btn-sm" hx-get="/projects/{{.Slug}}/edit" hx-target="#modal-content">Edit</button>
                        <button class="btn btn-danger btn-sm" hx-delete="/projects/{{.Slug}}" hx-target="closest tr" hx-confirm="Are you sure you want to delete this project?">Delete</button>
                    </div>
                </td>
            </tr>
            {{else}}
            <tr>
                <td colspan="4" class="px-6 py-8 text-center text-slate-500">No projects found</td>
            </tr>
            {{end}}
        </tbody>
    </table>
</div>
{{end}}`

const newProjectFormTemplate = `
<div class="px-6 py-5 border-b border-slate-200 flex justify-between items-center">
    <h3 class="text-lg font-semibold">New Project</h3>
    <button class="w-8 h-8 flex items-center justify-center rounded-lg text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition-all" onclick="document.getElementById('modal').style.display='none'">&times;</button>
</div>
<form hx-post="/projects" hx-target="#content" hx-on::after-request="if(event.detail.xhr.status >= 200 && event.detail.xhr.status < 300) document.getElementById('modal').style.display='none'">
    <input type="hidden" name="redirect_to" value="{{if .RedirectTo}}{{.RedirectTo}}{{else}}/instances{{end}}">
    <div class="p-6">
        <div id="form-error" class="mb-4"></div>
        <div class="mb-5">
            <label class="block text-sm font-medium mb-1.5" for="slug">Slug</label>
            <input type="text" id="slug" name="slug" placeholder="my-project" required pattern="[a-z][a-z0-9-]*" class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-[#2878B5] focus:ring-2 focus:ring-[#2878B5]/10 transition-all">
            <p class="text-xs text-slate-400 mt-1">Lowercase letters, numbers, and dashes. Must start with a letter.</p>
        </div>
        <div class="mb-5">
            <label class="block text-sm font-medium mb-1.5" for="name">Name</label>
            <input type="text" id="name" name="name" placeholder="My Project" required class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-[#2878B5] focus:ring-2 focus:ring-[#2878B5]/10 transition-all">
        </div>
    </div>
    <div class="px-6 py-4 border-t border-slate-200 flex justify-end gap-3 bg-slate-50">
        <button type="button" class="btn btn-secondary" onclick="document.getElementById('modal').style.display='none'">Cancel</button>
        <button type="submit" class="btn btn-primary">Create Project</button>
    </div>
</form>`

const editProjectFormTemplate = `
<div class="px-6 py-5 border-b border-slate-200 flex justify-between items-center">
    <h3 class="text-lg font-semibold">Edit Project</h3>
    <button class="w-8 h-8 flex items-center justify-center rounded-lg text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition-all" onclick="document.getElementById('modal').style.display='none'">&times;</button>
</div>
<form hx-put="/projects/{{.Project.Slug}}" hx-target="#content" hx-on::after-request="if(event.detail.xhr.status >= 200 && event.detail.xhr.status < 300) document.getElementById('modal').style.display='none'">
    <div class="p-6">
        <div id="form-error" class="mb-4"></div>
        <div class="mb-5">
            <label class="block text-sm font-medium mb-1.5" for="slug">Slug</label>
            <input type="text" id="slug" value="{{.Project.Slug}}" disabled class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg bg-slate-50 text-slate-500 cursor-not-allowed">
        </div>
        <div class="mb-5">
            <label class="block text-sm font-medium mb-1.5" for="name">Name</label>
            <input type="text" id="name" name="name" value="{{.Project.Name}}" required class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-[#2878B5] focus:ring-2 focus:ring-[#2878B5]/10 transition-all">
        </div>
    </div>
    <div class="px-6 py-4 border-t border-slate-200 flex justify-end gap-3 bg-slate-50">
        <button type="button" class="btn btn-secondary" onclick="document.getElementById('modal').style.display='none'">Cancel</button>
        <button type="submit" class="btn btn-primary">Update Project</button>
    </div>
</form>`

const instancesTemplate = `{{define "content"}}
<div class="overflow-hidden rounded-[28px] border border-black/5 bg-white/85 shadow-[0_18px_40px_rgba(15,23,42,0.06)] backdrop-blur">
    <div class="flex items-center justify-between border-b border-slate-200 px-6 py-5">
        <div>
            <h2 class="text-lg font-semibold">Instances</h2>
            {{if .Context.Project}}<p class="mt-1 text-sm text-slate-500">{{.Context.Project.Name}}</p>{{end}}
        </div>
        {{if .Context.Project}}
        <button class="btn btn-primary" hx-get="/projects/{{.Context.Project.Slug}}/instances/new" hx-target="#modal-content">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path>
            </svg>
            New Instance
        </button>
        {{end}}
    </div>
    {{if .Context.Project}}
    <table class="w-full">
        <thead>
            <tr>
                <th class="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wider text-slate-500 bg-slate-50 border-b border-slate-200">Name</th>
                <th class="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wider text-slate-500 bg-slate-50 border-b border-slate-200">Region</th>
                <th class="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wider text-slate-500 bg-slate-50 border-b border-slate-200">CPU</th>
                <th class="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wider text-slate-500 bg-slate-50 border-b border-slate-200">Memory</th>
                <th class="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wider text-slate-500 bg-slate-50 border-b border-slate-200">Image</th>
                <th class="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wider text-slate-500 bg-slate-50 border-b border-slate-200">Status</th>
                <th class="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wider text-slate-500 bg-slate-50 border-b border-slate-200">Actions</th>
            </tr>
        </thead>
        <tbody>
            {{range .Instances}}
            <tr class="hover:bg-slate-50">
                <td class="px-6 py-4 border-b border-slate-100">
                    <a href="#" hx-get="/projects/{{$.Context.Project.Slug}}/instances/{{.ID}}/edit" hx-target="#modal-content" class="font-medium text-[#2878B5] hover:underline">{{.Name}}</a>
                </td>
                <td class="px-6 py-4 border-b border-slate-100"><code class="bg-slate-100 px-2 py-0.5 rounded text-sm">{{.Region}}</code></td>
                <td class="px-6 py-4 border-b border-slate-100">{{.CPU}} vCPU</td>
                <td class="px-6 py-4 border-b border-slate-100">{{.MemoryMB}} MB</td>
                <td class="px-6 py-4 border-b border-slate-100"><code class="bg-slate-100 px-2 py-0.5 rounded text-sm">{{.Image}}</code></td>
                <td class="px-6 py-4 border-b border-slate-100">
                    {{if eq .Status "running"}}
                    <span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-emerald-50 text-emerald-600">
                        <span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span>
                        Running
                    </span>
                    {{else}}
                    <span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-red-50 text-red-600">
                        <span class="w-1.5 h-1.5 rounded-full bg-red-500"></span>
                        Stopped
                    </span>
                    {{end}}
                </td>
                <td class="px-6 py-4 border-b border-slate-100">
                    <div class="flex gap-2">
                        <button class="btn btn-secondary btn-sm" hx-get="/projects/{{$.Context.Project.Slug}}/instances/{{.ID}}/edit" hx-target="#modal-content">Edit</button>
                        <button class="btn btn-danger btn-sm" hx-delete="/projects/{{$.Context.Project.Slug}}/instances/{{.ID}}" hx-target="closest tr" hx-confirm="Are you sure you want to delete this instance?">Delete</button>
                    </div>
                </td>
            </tr>
            {{else}}
            <tr>
                <td colspan="7" class="px-6 py-8 text-center text-slate-500">No instances yet for this project.</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{else}}
    <div class="px-6 py-12 text-center">
        <p class="text-sm text-slate-500">Create a project to start adding instances.</p>
    </div>
    {{end}}
</div>
{{end}}`

const newInstanceFormTemplate = `
<div class="px-6 py-5 border-b border-slate-200 flex justify-between items-center">
    <h3 class="text-lg font-semibold">New Instance</h3>
    <button class="w-8 h-8 flex items-center justify-center rounded-lg text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition-all" onclick="document.getElementById('modal').style.display='none'">&times;</button>
</div>
<form hx-post="/projects/{{.Project.Slug}}/instances" hx-target="#content" hx-on::after-request="if(event.detail.xhr.status >= 200 && event.detail.xhr.status < 300) document.getElementById('modal').style.display='none'">
    <div class="p-6">
        <div id="form-error" class="mb-4"></div>
        <div class="mb-5">
            <label class="block text-sm font-medium mb-1.5" for="name">Instance Name</label>
            <input type="text" id="name" name="name" placeholder="my-instance" required class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-[#2878B5] focus:ring-2 focus:ring-[#2878B5]/10 transition-all">
        </div>
        <div class="mb-5">
            <label class="block text-sm font-medium mb-1.5" for="region">Region</label>
            <select id="region" name="region" required class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-[#2878B5] focus:ring-2 focus:ring-[#2878B5]/10 transition-all bg-white">
                {{range .Regions}}<option value="{{.}}">{{.}}</option>{{end}}
            </select>
        </div>
        <div class="grid grid-cols-2 gap-4 mb-5">
            <div>
                <label class="block text-sm font-medium mb-1.5" for="cpu">CPU (vCPU)</label>
                <input type="number" id="cpu" name="cpu" value="1" min="1" max="64" required class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-[#2878B5] focus:ring-2 focus:ring-[#2878B5]/10 transition-all">
            </div>
            <div>
                <label class="block text-sm font-medium mb-1.5" for="memory_mb">Memory (MB)</label>
                <input type="number" id="memory_mb" name="memory_mb" value="512" min="1" max="524288" required class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-[#2878B5] focus:ring-2 focus:ring-[#2878B5]/10 transition-all">
            </div>
        </div>
        <div class="mb-5">
            <label class="block text-sm font-medium mb-1.5" for="image">Image</label>
            <input type="text" id="image" name="image" value="ubuntu:20.04" required class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-[#2878B5] focus:ring-2 focus:ring-[#2878B5]/10 transition-all">
        </div>
        <div class="mb-5">
            <label class="block text-sm font-medium mb-1.5" for="status">Initial Status</label>
            <select id="status" name="status" class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-[#2878B5] focus:ring-2 focus:ring-[#2878B5]/10 transition-all bg-white">
                <option value="running">Running</option>
                <option value="stopped">Stopped</option>
            </select>
        </div>
    </div>
    <div class="px-6 py-4 border-t border-slate-200 flex justify-end gap-3 bg-slate-50">
        <button type="button" class="btn btn-secondary" onclick="document.getElementById('modal').style.display='none'">Cancel</button>
        <button type="submit" class="btn btn-primary">Create Instance</button>
    </div>
</form>`

const editInstanceFormTemplate = `
<div class="px-6 py-5 border-b border-slate-200 flex justify-between items-center">
    <h3 class="text-lg font-semibold">Edit Instance</h3>
    <button class="w-8 h-8 flex items-center justify-center rounded-lg text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition-all" onclick="document.getElementById('modal').style.display='none'">&times;</button>
</div>
<form hx-put="/projects/{{.Project.Slug}}/instances/{{.Instance.ID}}" hx-target="#content" hx-on::after-request="if(event.detail.xhr.status >= 200 && event.detail.xhr.status < 300) document.getElementById('modal').style.display='none'">
    <div class="p-6">
        <div id="form-error" class="mb-4"></div>
        <div class="mb-5">
            <label class="block text-sm font-medium mb-1.5" for="name">Instance Name</label>
            <input type="text" id="name" name="name" value="{{.Instance.Name}}" required class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-[#2878B5] focus:ring-2 focus:ring-[#2878B5]/10 transition-all">
        </div>
        <div class="mb-5">
            <label class="block text-sm font-medium mb-1.5" for="region">Region</label>
            <input type="text" id="region" value="{{.Instance.Region}}" disabled class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg bg-slate-50 text-slate-500 cursor-not-allowed">
            <p class="text-xs text-slate-400 mt-1">Region cannot be changed after creation</p>
        </div>
        <div class="grid grid-cols-2 gap-4 mb-5">
            <div>
                <label class="block text-sm font-medium mb-1.5" for="cpu">CPU (vCPU)</label>
                <input type="number" id="cpu" name="cpu" value="{{.Instance.CPU}}" min="1" max="64" required class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-[#2878B5] focus:ring-2 focus:ring-[#2878B5]/10 transition-all">
            </div>
            <div>
                <label class="block text-sm font-medium mb-1.5" for="memory_mb">Memory (MB)</label>
                <input type="number" id="memory_mb" name="memory_mb" value="{{.Instance.MemoryMB}}" min="1" max="524288" required class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-[#2878B5] focus:ring-2 focus:ring-[#2878B5]/10 transition-all">
            </div>
        </div>
        <div class="mb-5">
            <label class="block text-sm font-medium mb-1.5" for="image">Image</label>
            <input type="text" id="image" value="{{.Instance.Image}}" disabled class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg bg-slate-50 text-slate-500 cursor-not-allowed">
        </div>
        <div class="mb-5">
            <label class="block text-sm font-medium mb-1.5" for="status">Status</label>
            <select id="status" name="status" class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-[#2878B5] focus:ring-2 focus:ring-[#2878B5]/10 transition-all bg-white">
                <option value="running" {{if eq .Instance.Status "running"}}selected{{end}}>Running</option>
                <option value="stopped" {{if eq .Instance.Status "stopped"}}selected{{end}}>Stopped</option>
            </select>
        </div>
    </div>
    <div class="px-6 py-4 border-t border-slate-200 flex justify-end gap-3 bg-slate-50">
        <button type="button" class="btn btn-secondary" onclick="document.getElementById('modal').style.display='none'">Cancel</button>
        <button type="submit" class="btn btn-primary">Update Instance</button>
    </div>
</form>`

const metadataTemplate = `{{define "content"}}
<div class="bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden">
    <div class="px-6 py-5 border-b border-slate-200 flex justify-between items-center">
        <h2 class="text-lg font-semibold">Metadata</h2>
        <button class="btn btn-primary" hx-get="/metadata/new" hx-target="#modal-content">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path>
            </svg>
            New Metadata
        </button>
    </div>
    <div class="px-6 py-4 border-b border-slate-200">
        <div class="max-w-sm">
            <label class="block text-sm font-medium mb-1.5" for="prefix-filter">Filter by prefix</label>
            <input type="text" id="prefix-filter" name="prefix" hx-get="/metadata" hx-target="#content" hx-trigger="input changed delay:500ms" value="{{.Prefix}}" placeholder="Enter prefix to filter..." class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-[#2878B5] focus:ring-2 focus:ring-[#2878B5]/10 transition-all">
        </div>
    </div>
    <table class="w-full">
        <thead>
            <tr>
                <th class="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wider text-slate-500 bg-slate-50 border-b border-slate-200">Path</th>
                <th class="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wider text-slate-500 bg-slate-50 border-b border-slate-200">Value</th>
                <th class="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wider text-slate-500 bg-slate-50 border-b border-slate-200">Updated At</th>
                <th class="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wider text-slate-500 bg-slate-50 border-b border-slate-200">Actions</th>
            </tr>
        </thead>
        <tbody>
            {{range .Metadata}}
            <tr class="hover:bg-slate-50" id="row-{{.ID}}">
                <td class="px-6 py-4 border-b border-slate-100"><code class="bg-slate-100 px-2 py-0.5 rounded text-sm">{{.Path}}</code></td>
                <td class="px-6 py-4 border-b border-slate-100 max-w-xs truncate">{{.Value}}</td>
                <td class="px-6 py-4 border-b border-slate-100 text-slate-500">{{.UpdatedAt.Format "2006-01-02 15:04:05"}}</td>
                <td class="px-6 py-4 border-b border-slate-100">
                    <div class="flex gap-2">
                        <button class="btn btn-secondary btn-sm" hx-get="/metadata/edit?id={{.ID}}" hx-target="#modal-content">Edit</button>
                        <button class="btn btn-danger btn-sm" hx-delete="/metadata/delete?id={{.ID}}" hx-target="#row-{{.ID}}" hx-swap="outerHTML" hx-confirm="Are you sure you want to delete this metadata?">Delete</button>
                    </div>
                </td>
            </tr>
            {{else}}
            <tr>
                <td colspan="4" class="px-6 py-8 text-center text-slate-500">No metadata found</td>
            </tr>
            {{end}}
        </tbody>
    </table>
</div>
{{end}}`

const newMetadataFormTemplate = `
<div class="px-6 py-5 border-b border-slate-200 flex justify-between items-center">
    <h3 class="text-lg font-semibold">New Metadata</h3>
    <button class="w-8 h-8 flex items-center justify-center rounded-lg text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition-all" onclick="document.getElementById('modal').style.display='none'">&times;</button>
</div>
<form hx-post="/metadata" hx-target="#content" hx-on::after-request="if(event.detail.xhr.status >= 200 && event.detail.xhr.status < 300) document.getElementById('modal').style.display='none'">
    <div class="p-6">
        <div id="form-error" class="mb-4"></div>
        <div class="mb-5">
            <label class="block text-sm font-medium mb-1.5" for="path">Path</label>
            <input type="text" id="path" name="path" placeholder="config/settings/key" required class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-[#2878B5] focus:ring-2 focus:ring-[#2878B5]/10 transition-all">
        </div>
        <div class="mb-5">
            <label class="block text-sm font-medium mb-1.5" for="value">Value</label>
            <textarea id="value" name="value" rows="4" placeholder="Enter value..." required class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-[#2878B5] focus:ring-2 focus:ring-[#2878B5]/10 transition-all resize-none"></textarea>
        </div>
    </div>
    <div class="px-6 py-4 border-t border-slate-200 flex justify-end gap-3 bg-slate-50">
        <button type="button" class="btn btn-secondary" onclick="document.getElementById('modal').style.display='none'">Cancel</button>
        <button type="submit" class="btn btn-primary">Create Metadata</button>
    </div>
</form>`

const editMetadataFormTemplate = `
<div class="px-6 py-5 border-b border-slate-200 flex justify-between items-center">
    <h3 class="text-lg font-semibold">Edit Metadata</h3>
    <button class="w-8 h-8 flex items-center justify-center rounded-lg text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition-all" onclick="document.getElementById('modal').style.display='none'">&times;</button>
</div>
<form hx-put="/metadata/update" hx-target="#content" hx-on::after-request="if(event.detail.xhr.status >= 200 && event.detail.xhr.status < 300) document.getElementById('modal').style.display='none'">
    <input type="hidden" name="id" value="{{.Metadata.ID}}">
    <div class="p-6">
        <div id="form-error" class="mb-4"></div>
        <div class="mb-5">
            <label class="block text-sm font-medium mb-1.5" for="path">Path</label>
            <input type="text" id="path" name="path" value="{{.Metadata.Path}}" readonly class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg bg-slate-50 text-slate-500">
        </div>
        <div class="mb-5">
            <label class="block text-sm font-medium mb-1.5" for="value">Value</label>
            <textarea id="value" name="value" rows="4" required class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-[#2878B5] focus:ring-2 focus:ring-[#2878B5]/10 transition-all resize-none">{{.Metadata.Value}}</textarea>
        </div>
    </div>
    <div class="px-6 py-4 border-t border-slate-200 flex justify-end gap-3 bg-slate-50">
        <button type="button" class="btn btn-secondary" onclick="document.getElementById('modal').style.display='none'">Cancel</button>
        <button type="submit" class="btn btn-primary">Update Metadata</button>
    </div>
</form>`

const storageTemplate = `{{define "content"}}
<div class="overflow-hidden rounded-[28px] border border-black/5 bg-white/85 shadow-[0_18px_40px_rgba(15,23,42,0.06)] backdrop-blur">
    <div class="flex items-center justify-between border-b border-slate-200 px-6 py-5">
        <div>
            <h2 class="text-lg font-semibold">Storage Buckets</h2>
            {{if .Context.Project}}<p class="mt-1 text-sm text-slate-500">{{.Context.Project.Name}}</p>{{end}}
        </div>
        {{if .Context.Project}}
        <button class="btn btn-primary" hx-get="/projects/{{.Context.Project.Slug}}/storage/buckets/new" hx-target="#modal-content">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path>
            </svg>
            New Bucket
        </button>
        {{end}}
    </div>
    {{if .Context.Project}}
    <table class="w-full">
        <thead>
            <tr>
                <th class="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wider text-slate-500 bg-slate-50 border-b border-slate-200">Bucket Name</th>
                <th class="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wider text-slate-500 bg-slate-50 border-b border-slate-200">Created At</th>
                <th class="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wider text-slate-500 bg-slate-50 border-b border-slate-200">Updated At</th>
            </tr>
        </thead>
        <tbody>
            {{range .Buckets}}
            <tr class="hover:bg-slate-50">
                <td class="px-6 py-4 border-b border-slate-100">
                    <a href="/projects/{{$.Context.Project.Slug}}/storage/{{.Name}}" class="flex items-center gap-3 text-[#2878B5] hover:underline">
                        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4"></path>
                        </svg>
                        <span class="font-medium">{{.Name}}</span>
                    </a>
                </td>
                <td class="px-6 py-4 border-b border-slate-100 text-slate-500">{{.CreatedAt.Format "2006-01-02 15:04:05"}}</td>
                <td class="px-6 py-4 border-b border-slate-100 text-slate-500">{{.UpdatedAt.Format "2006-01-02 15:04:05"}}</td>
            </tr>
            {{else}}
            <tr>
                <td colspan="3" class="px-6 py-8 text-center text-slate-500">No buckets yet for this project.</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{else}}
    <div class="px-6 py-12 text-center">
        <p class="text-sm text-slate-500">Create a project to start storing objects.</p>
    </div>
    {{end}}
</div>
{{end}}`

const newBucketFormTemplate = `
<div class="px-6 py-5 border-b border-slate-200 flex justify-between items-center">
    <h3 class="text-lg font-semibold">New Bucket</h3>
    <button class="w-8 h-8 flex items-center justify-center rounded-lg text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition-all" onclick="document.getElementById('modal').style.display='none'">&times;</button>
</div>
<form hx-post="/projects/{{.Project.Slug}}/storage/buckets" hx-target="#content" hx-on::after-request="if(event.detail.xhr.status >= 200 && event.detail.xhr.status < 300) document.getElementById('modal').style.display='none'">
    <div class="p-6">
        <div id="form-error" class="mb-4"></div>
        <div class="mb-5">
            <label class="block text-sm font-medium mb-1.5" for="name">Bucket Name</label>
            <input type="text" id="name" name="name" placeholder="my-bucket" required class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-[#2878B5] focus:ring-2 focus:ring-[#2878B5]/10 transition-all">
        </div>
    </div>
    <div class="px-6 py-4 border-t border-slate-200 flex justify-end gap-3 bg-slate-50">
        <button type="button" class="btn btn-secondary" onclick="document.getElementById('modal').style.display='none'">Cancel</button>
        <button type="submit" class="btn btn-primary">Create Bucket</button>
    </div>
</form>`

const bucketObjectsTemplate = `{{define "content"}}
<div class="bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden">
    <div class="px-6 py-5 border-b border-slate-200 flex justify-between items-center">
        <div class="flex items-center gap-3">
            <a href="/storage" class="btn btn-secondary btn-sm">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"></path>
                </svg>
                Back
            </a>
            <h2 class="text-lg font-semibold">{{.Bucket.Name}}</h2>
        </div>
        <button class="btn btn-primary" hx-get="/projects/{{.Context.Project.Slug}}/storage/{{.Bucket.Name}}/objects/new" hx-target="#modal-content">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"></path>
            </svg>
            Upload Object
        </button>
    </div>
    <div class="px-6 py-4 border-b border-slate-200">
        <div class="max-w-sm">
            <label class="block text-sm font-medium mb-1.5" for="prefix-filter">Filter by prefix</label>
            <input type="text" id="prefix-filter" name="prefix" hx-get="/projects/{{.Context.Project.Slug}}/storage/{{.Bucket.Name}}" hx-params="*" hx-target="#content" hx-trigger="input changed delay:500ms" value="{{.Prefix}}" placeholder="folder/subfolder/" class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-[#2878B5] focus:ring-2 focus:ring-[#2878B5]/10 transition-all">
        </div>
    </div>
    <table class="w-full">
        <thead>
            <tr>
                <th class="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wider text-slate-500 bg-slate-50 border-b border-slate-200">Path</th>
                <th class="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wider text-slate-500 bg-slate-50 border-b border-slate-200">Updated At</th>
                <th class="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wider text-slate-500 bg-slate-50 border-b border-slate-200">Actions</th>
            </tr>
        </thead>
        <tbody>
            {{range .Objects}}
            <tr class="hover:bg-slate-50">
                <td class="px-6 py-4 border-b border-slate-100">
                    <div class="flex items-center gap-3">
                        <svg class="w-4 h-4 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"></path>
                        </svg>
                        <code class="bg-slate-100 px-2 py-0.5 rounded text-sm">{{.Path}}</code>
                    </div>
                </td>
                <td class="px-6 py-4 border-b border-slate-100 text-slate-500">{{.UpdatedAt.Format "2006-01-02 15:04:05"}}</td>
                <td class="px-6 py-4 border-b border-slate-100">
                    <button class="btn btn-secondary btn-sm" hx-get="/projects/{{$.Context.Project.Slug}}/storage/{{$.Bucket.Name}}/objects/{{.ID}}" hx-target="#modal-content">View</button>
                </td>
            </tr>
            {{else}}
            <tr>
                <td colspan="3" class="px-6 py-8 text-center text-slate-500">No objects found</td>
            </tr>
            {{end}}
        </tbody>
    </table>
</div>
{{end}}`

const newObjectFormTemplate = `
<div class="px-6 py-5 border-b border-slate-200 flex justify-between items-center">
    <h3 class="text-lg font-semibold">Upload Object to {{.Bucket.Name}}</h3>
    <button class="w-8 h-8 flex items-center justify-center rounded-lg text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition-all" onclick="document.getElementById('modal').style.display='none'">&times;</button>
</div>
<form hx-post="/projects/{{.Project.Slug}}/storage/{{.Bucket.Name}}/objects" hx-target="#content" hx-on::after-request="if(event.detail.xhr.status >= 200 && event.detail.xhr.status < 300) document.getElementById('modal').style.display='none'">
    <div class="p-6">
        <div id="form-error" class="mb-4"></div>
        <div class="mb-5">
            <label class="block text-sm font-medium mb-1.5" for="path">Object Path</label>
            <input type="text" id="path" name="path" placeholder="folder/file.txt" required class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-[#2878B5] focus:ring-2 focus:ring-[#2878B5]/10 transition-all">
        </div>
        <div class="mb-5">
            <label class="block text-sm font-medium mb-1.5" for="content">Content</label>
            <textarea id="content" name="content" rows="8" placeholder="Enter file content..." required class="w-full px-3.5 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-[#2878B5] focus:ring-2 focus:ring-[#2878B5]/10 transition-all resize-none font-mono"></textarea>
        </div>
        <p class="text-xs text-slate-500">The content will be base64-encoded and stored.</p>
    </div>
    <div class="px-6 py-4 border-t border-slate-200 flex justify-end gap-3 bg-slate-50">
        <button type="button" class="btn btn-secondary" onclick="document.getElementById('modal').style.display='none'">Cancel</button>
        <button type="submit" class="btn btn-primary">Upload Object</button>
    </div>
</form>`

const viewObjectTemplate = `
<div class="px-6 py-5 border-b border-slate-200 flex justify-between items-center">
    <h3 class="text-lg font-semibold">{{.Object.Path}}</h3>
    <button class="w-8 h-8 flex items-center justify-center rounded-lg text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition-all" onclick="document.getElementById('modal').style.display='none'">&times;</button>
</div>
<div class="p-6">
    <div class="flex gap-6 mb-4">
        <div>
            <span class="block text-xs uppercase tracking-wider text-slate-500 mb-1">Size</span>
            <span class="font-medium">{{.Size}} bytes</span>
        </div>
    </div>
    <div>
        <span class="block text-xs uppercase tracking-wider text-slate-500 mb-2">Content</span>
        <pre class="bg-slate-50 border border-slate-200 rounded-lg p-4 overflow-auto max-h-[50vh] text-sm font-mono whitespace-pre-wrap break-words">{{.DecodedContent}}</pre>
    </div>
</div>
<div class="px-6 py-4 border-t border-slate-200 flex justify-end bg-slate-50">
    <button class="btn btn-secondary" onclick="document.getElementById('modal').style.display='none'">Close</button>
</div>`

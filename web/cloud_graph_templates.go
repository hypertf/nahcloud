package web

const cloudGraphTemplate = `{{define "content"}}
<div class="space-y-6">
    <header class="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
        <div>
            <div class="flex items-center gap-3">
                <p class="text-xs font-semibold uppercase tracking-[0.22em] text-[#2f6db5]">Project topology</p>
                <span class="rounded-full border border-amber-200 bg-amber-50 px-2.5 py-1 text-[10px] font-semibold uppercase tracking-wider text-amber-700">Preview adapter</span>
            </div>
            <h2 class="mt-2 text-4xl font-semibold tracking-tight text-slate-950">Cloud graph</h2>
            {{if .Context.Project}}<p class="mt-3 max-w-2xl text-sm leading-6 text-slate-600">Relationships in <strong>{{.Context.Project.Name}}</strong>, arranged by ownership and traffic flow instead of isolated inventory tables.</p>{{end}}
        </div>
        {{if .Context.Project}}
        <div class="flex flex-wrap gap-2">
            <button class="btn btn-secondary" hx-get="/projects/{{.Context.Project.Slug}}/cloud/new?kind=network" hx-target="#modal-content">New network</button>
            <button class="btn btn-secondary" hx-get="/projects/{{.Context.Project.Slug}}/cloud/new?kind=disk" hx-target="#modal-content">New disk</button>
            <button class="btn btn-primary" hx-get="/projects/{{.Context.Project.Slug}}/cloud/new?kind=load-balancer" hx-target="#modal-content">New load balancer</button>
        </div>
        {{end}}
    </header>

    {{if not .Context.Project}}
    <section class="console-card rounded-[28px] p-10 text-center">
        <div class="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-blue-50 text-2xl">⌘</div>
        <h3 class="mt-5 text-xl font-semibold">Create a project to map your cloud</h3>
        <p class="mx-auto mt-2 max-w-md text-sm leading-6 text-slate-600">Networks, disks, policies, load balancers, and fault scenarios are isolated within a project.</p>
        <button class="btn btn-primary mt-5" hx-get="/projects/new?redirect_to=/cloud" hx-target="#modal-content">Create project</button>
    </section>
    {{else if eq .State "loading"}}
    <section aria-label="Loading cloud graph" aria-busy="true" class="grid gap-5 lg:grid-cols-2">
        <div class="h-72 animate-pulse rounded-[28px] border border-slate-200 bg-white/70 p-6"><div class="h-4 w-28 rounded bg-slate-200"></div><div class="mt-8 h-20 rounded-2xl bg-slate-100"></div><div class="mt-4 h-20 rounded-2xl bg-slate-100"></div></div>
        <div class="h-72 animate-pulse rounded-[28px] border border-slate-200 bg-white/70 p-6"><div class="h-4 w-36 rounded bg-slate-200"></div><div class="mt-8 h-44 rounded-2xl bg-slate-100"></div></div>
    </section>
    {{else if eq .State "error"}}
    <section role="alert" class="rounded-[28px] border border-red-200 bg-red-50 p-8">
        <p class="text-xs font-bold uppercase tracking-[0.2em] text-red-600">Graph unavailable</p>
        <h3 class="mt-2 text-xl font-semibold text-slate-950">We could not load this project topology</h3>
        <p class="mt-2 text-sm text-slate-700">{{.Message}}</p>
        <a href="/cloud" class="btn btn-secondary mt-5">Try again</a>
    </section>
    {{else if eq .State "empty"}}
    <section class="console-card rounded-[28px] p-10 text-center">
        <div class="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-slate-100 text-2xl text-slate-500">◇</div>
        <h3 class="mt-5 text-xl font-semibold">This graph is empty</h3>
        <p class="mx-auto mt-2 max-w-md text-sm leading-6 text-slate-600">Start with a network. Subnets and policy bindings will appear nested beneath their parent resources.</p>
        <button class="btn btn-primary mt-5" hx-get="/projects/{{.Context.Project.Slug}}/cloud/new?kind=network" hx-target="#modal-content">Create first network</button>
    </section>
    {{else}}
    <div class="grid gap-5 xl:grid-cols-2">
        <section id="networks" class="console-card rounded-[28px] p-6">
            <div class="flex items-start justify-between gap-4">
                <div><p class="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">Network fabric</p><h3 class="mt-1 text-xl font-semibold">Networks → subnets</h3></div>
                <button class="text-sm font-semibold text-[#2f6db5]" hx-get="/projects/{{.Context.Project.Slug}}/cloud/new?kind=subnet" hx-target="#modal-content">+ Subnet</button>
            </div>
            <div class="mt-5 space-y-4">
                {{range .Graph.Networks}}
                <article id="network-{{.ID}}" class="rounded-2xl border border-slate-200 bg-white p-4">
                    <div class="flex items-start justify-between gap-3">
                        <div><div class="flex items-center gap-2"><span class="h-2.5 w-2.5 rounded-full bg-blue-500"></span><h4 class="font-semibold">{{.Name}}</h4></div><p class="mt-1 font-mono text-xs text-slate-500">{{.CIDR}} · {{.Region}}</p></div>
                        <button aria-label="Delete network {{.Name}}" class="text-xs font-medium text-red-600" hx-delete="/projects/{{$.Context.Project.Slug}}/cloud/network/{{.ID}}" hx-target="#content" hx-confirm="Delete network {{.Name}} and its child subnets?">Delete</button>
                    </div>
                    <div class="ml-1 mt-4 border-l-2 border-blue-100 pl-4">
                        {{range .Subnets}}
                        <div id="subnet-{{.ID}}" class="mb-2 flex items-center justify-between rounded-xl bg-slate-50 px-3 py-2.5 last:mb-0">
                            <div><p class="text-sm font-medium">{{.Name}}</p><p class="font-mono text-[11px] text-slate-500">{{.CIDR}} · {{.Zone}}</p></div>
                            <button aria-label="Delete subnet {{.Name}}" class="text-xs text-red-600" hx-delete="/projects/{{$.Context.Project.Slug}}/cloud/subnet/{{.ID}}" hx-target="#content" hx-confirm="Delete subnet {{.Name}}?">Remove</button>
                        </div>
                        {{else}}<p class="py-3 text-xs text-slate-500">No child subnets.</p>{{end}}
                    </div>
                </article>
                {{else}}<p class="rounded-2xl bg-slate-50 p-5 text-sm text-slate-500">No networks yet.</p>{{end}}
            </div>
        </section>

        <section id="load-balancers" class="console-card rounded-[28px] p-6">
            <div class="flex items-start justify-between gap-4">
                <div><p class="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">Traffic path</p><h3 class="mt-1 text-xl font-semibold">Load balancers → backends</h3></div>
                <button class="text-sm font-semibold text-[#2f6db5]" hx-get="/projects/{{.Context.Project.Slug}}/cloud/new?kind=backend" hx-target="#modal-content">+ Backend</button>
            </div>
            <div class="mt-5 space-y-4">
                {{range .Graph.LoadBalancers}}
                <article id="load-balancer-{{.ID}}" class="rounded-2xl border border-slate-200 bg-white p-4">
                    <div class="flex items-start justify-between gap-3"><div><h4 class="font-semibold">{{.Name}}</h4><p class="mt-1 font-mono text-xs text-slate-500">{{.Address}} · {{.Protocol}}</p></div><button class="text-xs font-medium text-red-600" hx-delete="/projects/{{$.Context.Project.Slug}}/cloud/load-balancer/{{.ID}}" hx-target="#content" hx-confirm="Delete load balancer {{.Name}} and detach all backends?">Delete</button></div>
                    <div class="mt-4 flex items-center gap-2 text-[10px] font-semibold uppercase tracking-wider text-slate-400"><span>Listener</span><span>→</span><span>{{len .Backends}} backends</span></div>
                    <div class="mt-2 grid gap-2 sm:grid-cols-2">
                        {{range .Backends}}
                        <div id="backend-{{.ID}}" class="rounded-xl bg-slate-50 p-3"><div class="flex items-center justify-between"><a href="#instance-{{.InstanceID}}" class="text-sm font-semibold text-[#2f6db5] hover:underline">{{.InstanceName}}:{{.Port}}</a><span class="rounded-full bg-emerald-100 px-2 py-0.5 text-[10px] font-semibold text-emerald-700">{{.Health}}</span></div><button class="mt-2 text-[11px] text-red-600" hx-delete="/projects/{{$.Context.Project.Slug}}/cloud/backend/{{.ID}}" hx-target="#content" hx-confirm="Detach {{.InstanceName}} from this load balancer?">Detach</button></div>
                        {{else}}<p class="text-xs text-slate-500">No backends attached.</p>{{end}}
                    </div>
                </article>
                {{else}}<p class="rounded-2xl bg-slate-50 p-5 text-sm text-slate-500">No load balancers yet.</p>{{end}}
            </div>
        </section>

        <section id="disks" class="console-card rounded-[28px] p-6">
            <div class="flex items-start justify-between gap-4"><div><p class="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">Block storage</p><h3 class="mt-1 text-xl font-semibold">Disks ↔ attachments</h3></div><button class="text-sm font-semibold text-[#2f6db5]" hx-get="/projects/{{.Context.Project.Slug}}/cloud/new?kind=disk" hx-target="#modal-content">+ Disk</button></div>
            <div class="mt-5 grid gap-3 sm:grid-cols-2">
                {{range .Graph.Disks}}
                <article id="disk-{{.ID}}" class="rounded-2xl border border-slate-200 bg-white p-4"><div class="flex justify-between gap-3"><div><h4 class="font-semibold">{{.Name}}</h4><p class="mt-1 text-xs text-slate-500">{{.Size}} · {{.Region}}</p></div><button class="text-xs text-red-600" hx-delete="/projects/{{$.Context.Project.Slug}}/cloud/disk/{{.ID}}" hx-target="#content" hx-confirm="Delete disk {{.Name}}? Attached data will be lost.">Delete</button></div>{{if .InstanceID}}<div class="mt-4 rounded-xl bg-blue-50 p-3"><p class="text-[10px] font-semibold uppercase tracking-wider text-blue-600">Attached to</p><a href="#instance-{{.InstanceID}}" class="mt-1 block text-sm font-semibold text-blue-900">{{.InstanceName}}</a><p class="font-mono text-[11px] text-blue-700">{{.MountPath}}</p></div>{{else}}<p class="mt-4 rounded-xl bg-slate-50 p-3 text-xs text-slate-500">Unattached</p>{{end}}</article>
                {{else}}<p class="text-sm text-slate-500">No disks yet.</p>{{end}}
            </div>
        </section>

        <section id="policies" class="console-card rounded-[28px] p-6">
            <div class="flex items-start justify-between gap-4"><div><p class="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">Access graph</p><h3 class="mt-1 text-xl font-semibold">Policies → bindings</h3></div><div class="flex gap-3"><button class="text-sm font-semibold text-[#2f6db5]" hx-get="/projects/{{.Context.Project.Slug}}/cloud/new?kind=policy" hx-target="#modal-content">+ Policy</button><button class="text-sm font-semibold text-[#2f6db5]" hx-get="/projects/{{.Context.Project.Slug}}/cloud/new?kind=binding" hx-target="#modal-content">+ Binding</button></div></div>
            <div class="mt-5 space-y-3">
                {{range .Graph.Policies}}
                <article id="policy-{{.ID}}" class="rounded-2xl border border-slate-200 bg-white p-4"><div class="flex justify-between"><div><h4 class="font-semibold">{{.Name}}</h4><span class="mt-1 inline-block rounded bg-emerald-50 px-2 py-0.5 text-[10px] font-bold uppercase text-emerald-700">{{.Effect}}</span></div><button class="text-xs text-red-600" hx-delete="/projects/{{$.Context.Project.Slug}}/cloud/policy/{{.ID}}" hx-target="#content" hx-confirm="Delete policy {{.Name}} and its bindings?">Delete</button></div><div class="mt-3 space-y-2">{{range .Bindings}}<div class="flex items-center justify-between rounded-xl bg-slate-50 p-3"><div><p class="text-sm"><strong>{{.Principal}}</strong> as {{.Role}}</p><a href="#{{.TargetKind}}-{{.TargetID}}" class="text-xs font-medium text-[#2f6db5]">→ {{.TargetKind}} / {{.TargetName}}</a></div><button class="text-[11px] text-red-600" hx-delete="/projects/{{$.Context.Project.Slug}}/cloud/binding/{{.ID}}" hx-target="#content" hx-confirm="Remove this binding?">Remove</button></div>{{else}}<p class="text-xs text-slate-500">No bindings.</p>{{end}}</div></article>
                {{else}}<p class="text-sm text-slate-500">No policies yet.</p>{{end}}
            </div>
        </section>
    </div>

    <section id="instances" class="rounded-[28px] border border-slate-800 bg-slate-950 p-6 text-white shadow-xl">
        <p class="text-xs font-semibold uppercase tracking-[0.2em] text-slate-400">Relationship index</p><h3 class="mt-1 text-xl font-semibold">Compute targets</h3>
        <div class="mt-5 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">{{range .Graph.Instances}}<article id="instance-{{.ID}}" class="rounded-2xl border border-slate-700 bg-slate-900 p-4"><div class="flex items-center gap-2"><span class="h-2 w-2 rounded-full bg-emerald-400"></span><h4 class="font-semibold">{{.Name}}</h4></div><p class="mt-2 font-mono text-xs text-slate-400">{{.ID}}</p><p class="mt-1 text-xs text-slate-500">{{.Region}}</p></article>{{end}}</div>
    </section>

    <section id="fault-tools" class="rounded-[28px] border-2 border-dashed border-amber-300 bg-amber-50/80 p-6">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between"><div><div class="flex items-center gap-2"><span class="rounded-md bg-amber-200 px-2 py-1 text-[10px] font-bold uppercase tracking-wider text-amber-900">Testing tools</span><h3 class="font-semibold text-slate-950">Tenant fault scenarios</h3></div><p class="mt-2 max-w-2xl text-sm text-slate-600">Simulated failures are scoped to this project. They are not lifecycle controls and never affect another tenant.</p></div></div>
        <div class="mt-5 grid gap-3 lg:grid-cols-3">{{range .Graph.Faults}}<article class="rounded-2xl border {{if .Enabled}}border-red-300 bg-red-50{{else}}border-amber-200 bg-white{{end}} p-4"><div class="flex items-start justify-between gap-3"><div><h4 class="text-sm font-semibold">{{.Label}}</h4><p class="mt-1 text-xs leading-5 text-slate-600">{{.Description}}</p></div><form hx-post="/projects/{{$.Context.Project.Slug}}/cloud/faults/{{.Kind}}" hx-target="#content"><input type="hidden" name="enabled" value="{{if .Enabled}}false{{else}}true{{end}}"><button class="rounded-lg px-3 py-1.5 text-xs font-semibold {{if .Enabled}}bg-red-600 text-white{{else}}bg-amber-200 text-amber-950{{end}}" type="submit" {{if .Enabled}}hx-confirm="Stop this fault scenario?"{{else}}hx-confirm="Start {{.Label}} for this project only?"{{end}}>{{if .Enabled}}Stop{{else}}Inject{{end}}</button></form></div>{{if .Enabled}}<p class="mt-3 text-[10px] font-bold uppercase tracking-wider text-red-700">Active fault</p>{{end}}</article>{{end}}</div>
    </section>
    {{end}}

    {{if .Context.Project}}<footer class="flex flex-wrap items-center gap-3 text-xs text-slate-500"><span>Preview states:</span><a class="hover:text-slate-900" href="/cloud?state=loading">Loading</a><a class="hover:text-slate-900" href="/cloud?state=empty">Empty</a><a class="hover:text-slate-900" href="/cloud?state=error">Error</a><a class="font-medium text-[#2f6db5]" href="/cloud">Live preview</a></footer>{{end}}
</div>
{{end}}`

const cloudGraphFormTemplate = `<div class="p-6">
    <div class="flex items-start justify-between gap-4"><div><p class="text-xs font-semibold uppercase tracking-[0.2em] text-[#2f6db5]">Cloud graph</p><h3 class="mt-1 text-xl font-semibold capitalize">Create {{.Kind}}</h3></div><button type="button" class="text-2xl text-slate-400" onclick="document.getElementById('modal').style.display='none'">×</button></div>
    <form class="mt-5 space-y-4" hx-post="/projects/{{.Project.Slug}}/cloud/resources" hx-target="#content">
        <input type="hidden" name="kind" value="{{.Kind}}"><div id="form-error"></div>
        {{if or (eq .Kind "network") (eq .Kind "disk") (eq .Kind "policy") (eq .Kind "load-balancer")}}
        <label class="block text-sm font-medium">Name<input name="name" required placeholder="{{if eq .Kind "network"}}production{{else if eq .Kind "disk"}}api-data{{else if eq .Kind "policy"}}service-operators{{else}}public-api{{end}}" class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5"></label>
        {{end}}
        {{if eq .Kind "network"}}<div class="grid grid-cols-2 gap-3"><label class="text-sm font-medium">CIDR<input name="cidr" required value="10.0.0.0/16" class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5 font-mono"></label><label class="text-sm font-medium">Region<input name="region" required value="us-east-1" class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5"></label></div>{{end}}
        {{if eq .Kind "subnet"}}<label class="block text-sm font-medium">Parent network<select name="parent_id" required class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5">{{range .Graph.Networks}}<option value="{{.ID}}">{{.Name}} · {{.CIDR}}</option>{{end}}</select></label><label class="block text-sm font-medium">Name<input name="name" required placeholder="private-b" class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5"></label><div class="grid grid-cols-2 gap-3"><label class="text-sm font-medium">CIDR<input name="cidr" required value="10.0.2.0/24" class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5 font-mono"></label><label class="text-sm font-medium">Zone<input name="zone" required value="us-east-1b" class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5"></label></div>{{end}}
        {{if eq .Kind "disk"}}<div class="grid grid-cols-2 gap-3"><label class="text-sm font-medium">Size<input name="size" required value="100 GiB" class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5"></label><label class="text-sm font-medium">Region<input name="region" required value="us-east-1" class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5"></label></div><label class="block text-sm font-medium">Attach to (optional)<select name="instance_id" class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5"><option value="">Leave unattached</option>{{range .Graph.Instances}}<option value="{{.ID}}">{{.Name}} · {{.Region}}</option>{{end}}</select></label><label class="block text-sm font-medium">Mount path<input name="mount_path" value="/data" class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5 font-mono"></label>{{end}}
        {{if eq .Kind "policy"}}<label class="block text-sm font-medium">Effect<select name="effect" class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5"><option value="allow">Allow</option><option value="deny">Deny</option></select></label>{{end}}
        {{if eq .Kind "binding"}}<label class="block text-sm font-medium">Policy<select name="parent_id" required class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5">{{range .Graph.Policies}}<option value="{{.ID}}">{{.Name}}</option>{{end}}</select></label><div class="grid grid-cols-2 gap-3"><label class="text-sm font-medium">Principal<input name="principal" required value="team:platform" class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5"></label><label class="text-sm font-medium">Role<input name="role" required value="viewer" class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5"></label></div><label class="block text-sm font-medium">Target<select name="target_ref" required class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5"><optgroup label="Networks">{{range .Graph.Networks}}<option value="network:{{.ID}}">{{.Name}}</option>{{end}}</optgroup><optgroup label="Disks">{{range .Graph.Disks}}<option value="disk:{{.ID}}">{{.Name}}</option>{{end}}</optgroup><optgroup label="Load balancers">{{range .Graph.LoadBalancers}}<option value="load-balancer:{{.ID}}">{{.Name}}</option>{{end}}</optgroup><optgroup label="Instances">{{range .Graph.Instances}}<option value="instance:{{.ID}}">{{.Name}}</option>{{end}}</optgroup></select></label>{{end}}
        {{if eq .Kind "load-balancer"}}<div class="grid grid-cols-2 gap-3"><label class="text-sm font-medium">Address<input name="address" required value="203.0.113.50" class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5 font-mono"></label><label class="text-sm font-medium">Listener<input name="protocol" required value="HTTPS :443" class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5"></label></div>{{end}}
        {{if eq .Kind "backend"}}<label class="block text-sm font-medium">Load balancer<select name="parent_id" required class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5">{{range .Graph.LoadBalancers}}<option value="{{.ID}}">{{.Name}}</option>{{end}}</select></label><div class="grid grid-cols-2 gap-3"><label class="text-sm font-medium">Instance<select name="instance_id" required class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5">{{range .Graph.Instances}}<option value="{{.ID}}">{{.Name}}</option>{{end}}</select></label><label class="text-sm font-medium">Port<input name="port" required value="8080" class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5 font-mono"></label></div>{{end}}
        <div class="flex justify-end gap-2 border-t border-slate-100 pt-4"><button type="button" class="btn btn-secondary" onclick="document.getElementById('modal').style.display='none'">Cancel</button><button type="submit" class="btn btn-primary">Create</button></div>
    </form>
</div>`

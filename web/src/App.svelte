<script lang="ts">
  // GET /api/ports のレスポンス（internal/server の PortView に対応）
  type PortView = {
    port: number;
    proto: string;
    address: string;
    pid: number;
    process: string;
    name?: string;
    note?: string;
    active: boolean;
  };

  let ports = $state<PortView[]>([]);
  let error = $state("");
  let editing = $state<number | null>(null); // 編集中のポート番号 or null
  let name = $state("");
  let note = $state("");

  async function load() {
    try {
      const res = await fetch("/api/ports");
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      ports = await res.json();
      error = "";
    } catch (e) {
      error = `ポート一覧の取得に失敗しました: ${e instanceof Error ? e.message : String(e)}`;
    }
  }

  function startEdit(p: PortView) {
    editing = p.port;
    name = p.name ?? "";
    note = p.note ?? "";
  }

  async function save() {
    const res = await fetch(`/api/ports/${editing}/label`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, note }),
    });
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      error = body.error ?? `保存に失敗しました (HTTP ${res.status})`;
      return;
    }
    editing = null;
    await load();
  }

  async function clearLabel(port: number) {
    await fetch(`/api/ports/${port}/label`, { method: "DELETE" });
    await load();
  }

  $effect(() => {
    load();
    const timer = setInterval(() => {
      if (editing === null) load();
    }, 5000);
    return () => clearInterval(timer);
  });
</script>

<h1>127xx<span class="sub">port registry</span></h1>

{#if error}<div class="error">{error}</div>{/if}

<table>
  <thead>
    <tr>
      <th>Port</th><th>Name</th><th>Process</th><th>PID</th><th>Address</th><th></th>
    </tr>
  </thead>
  <tbody>
    {#each ports as p (p.port + "|" + p.address)}
      <tr class:inactive={!p.active}>
        <td>{p.port}</td>
        <td>
          {#if editing === p.port}
            <input placeholder="名前" bind:value={name} />
            <input placeholder="メモ" bind:value={note} />
            <button onclick={save}>保存</button>
            <button onclick={() => (editing = null)}>取消</button>
          {:else}
            <span class="name">{p.name ?? ""}</span>
            {#if p.note}<div class="note">{p.note}</div>{/if}
          {/if}
        </td>
        <td>{p.active ? p.process : "(stopped)"}</td>
        <td>{p.active && p.pid > 0 ? p.pid : ""}</td>
        <td>{p.address}</td>
        <td>
          {#if editing !== p.port}
            <button onclick={() => startEdit(p)}>{p.name ? "編集" : "名前を付ける"}</button>
            {#if p.name}<button onclick={() => clearLabel(p.port)}>削除</button>{/if}
          {/if}
        </td>
      </tr>
    {/each}
  </tbody>
</table>

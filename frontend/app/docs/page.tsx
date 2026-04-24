import Link from "next/link";

export default function DocsPage() {
  return (
    <article className="mx-auto max-w-3xl px-4 py-8 prose">
      <h1>Documentation</h1>
      <ul>
        <li><Link href="/docs/api">REST + GraphQL API</Link></li>
        <li><Link href="/docs/mcp">MCP server</Link></li>
        <li><Link href="/docs/byok">BYOK setup</Link></li>
        <li><a href="https://github.com/evidencelens/evidencelens">Source on GitHub</a></li>
      </ul>
    </article>
  );
}

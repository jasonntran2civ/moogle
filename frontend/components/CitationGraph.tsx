"use client";

import { useEffect, useRef, useState } from "react";
import * as d3 from "d3";

interface Neighbor { id: string; title: string; pagerank?: number }
interface Edge { source: string; target: string }
interface GraphResp { nodes: Neighbor[]; edges: Edge[] }

const GATEWAY_URL = process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8080";

/**
 * Citation neighborhood viz (spec §8). Calls the gateway tool endpoint
 * `get_citation_neighborhood`, then renders a force-directed graph with
 * d3-force. The seed document is highlighted; node radius scales with
 * citation pagerank; edges are arrows from citing to cited.
 */
export function CitationGraph({ documentId, depth = 1 }: { documentId: string; depth?: number }) {
  const svgRef = useRef<SVGSVGElement | null>(null);
  const [graph, setGraph] = useState<GraphResp | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setError(null);
    fetch(`${GATEWAY_URL}/api/tool/get_citation_neighborhood`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ id: documentId, depth, top_k_per_hop: 30 }),
    })
      .then(r => (r.ok ? r.json() : Promise.reject(`HTTP ${r.status}`)))
      .then((d: GraphResp) => { if (!cancelled) setGraph(d); })
      .catch(e => { if (!cancelled) setError(String(e)); });
    return () => { cancelled = true; };
  }, [documentId, depth]);

  useEffect(() => {
    if (!graph || !svgRef.current) return;
    const width = 720;
    const height = 420;
    const svg = d3.select(svgRef.current).attr("viewBox", `0 0 ${width} ${height}`);
    svg.selectAll("*").remove();

    const nodes = graph.nodes.map(n => ({ ...n })) as any[];
    const links = graph.edges.map(e => ({ source: e.source, target: e.target })) as any[];

    svg.append("defs").append("marker")
      .attr("id", "arrow").attr("viewBox", "0 -5 10 10")
      .attr("refX", 14).attr("refY", 0).attr("markerWidth", 6).attr("markerHeight", 6)
      .attr("orient", "auto")
      .append("path").attr("d", "M0,-5L10,0L0,5").attr("fill", "currentColor").attr("opacity", 0.4);

    const sim = d3.forceSimulation(nodes)
      .force("link", d3.forceLink(links).id((d: any) => d.id).distance(60))
      .force("charge", d3.forceManyBody().strength(-180))
      .force("center", d3.forceCenter(width / 2, height / 2));

    const link = svg.append("g").attr("stroke", "currentColor").attr("opacity", 0.4)
      .selectAll("line").data(links).join("line").attr("marker-end", "url(#arrow)");

    const node = svg.append("g")
      .selectAll("circle").data(nodes).join("circle")
      .attr("r", (d: any) => 4 + Math.min(8, Math.log1p((d.pagerank ?? 0) * 10000) * 2))
      .attr("fill", (d: any) => d.id === documentId ? "hsl(var(--accent))" : "hsl(var(--coi))")
      .attr("stroke", "white").attr("stroke-width", 1)
      .call(d3.drag<SVGCircleElement, any>()
        .on("start", (event, d) => { if (!event.active) sim.alphaTarget(0.3).restart(); d.fx = d.x; d.fy = d.y; })
        .on("drag",  (event, d) => { d.fx = event.x; d.fy = event.y; })
        .on("end",   (event, d) => { if (!event.active) sim.alphaTarget(0); d.fx = null; d.fy = null; }));

    node.append("title").text((d: any) => d.title || d.id);

    sim.on("tick", () => {
      link.attr("x1", (d: any) => d.source.x).attr("y1", (d: any) => d.source.y)
        .attr("x2", (d: any) => d.target.x).attr("y2", (d: any) => d.target.y);
      node.attr("cx", (d: any) => d.x).attr("cy", (d: any) => d.y);
    });

    return () => { sim.stop(); };
  }, [graph, documentId]);

  if (error) {
    return <p className="text-sm text-[hsl(var(--muted))]">Could not load citation graph: {error}</p>;
  }
  if (!graph) {
    return <p role="status" aria-live="polite" className="text-sm text-[hsl(var(--muted))]">Loading citation graph…</p>;
  }
  return (
    <figure aria-label={`Citation graph for ${documentId}`} className="border rounded">
      <svg ref={svgRef} role="img" aria-hidden="false" className="w-full h-auto" />
      <figcaption className="sr-only">{graph.nodes.length} nodes, {graph.edges.length} edges</figcaption>
    </figure>
  );
}

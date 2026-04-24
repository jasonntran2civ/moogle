export default async function TrialPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      <h1 className="text-2xl font-semibold">Trial {id}</h1>
      <p className="text-sm text-[hsl(var(--muted))]">TODO: render Trial details from /api/trials.</p>
    </div>
  );
}

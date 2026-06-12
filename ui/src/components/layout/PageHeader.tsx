interface PageHeaderProps {
  title: string;
  subtitle?: string;
}

/** Page-level heading block. Presentational and reusable across screens. */
export function PageHeader({ title, subtitle }: PageHeaderProps) {
  return (
    <header className="mb-8 space-y-1">
      <h1 className="text-3xl font-bold tracking-tight">{title}</h1>
      {subtitle && <p className="text-muted-foreground">{subtitle}</p>}
    </header>
  );
}

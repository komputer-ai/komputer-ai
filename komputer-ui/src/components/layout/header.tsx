interface HeaderProps {
  title: string;
  children?: React.ReactNode;
}

export function Header({ title, children }: HeaderProps) {
  return (
    <header className="flex items-center justify-between px-6 h-14 border-b border-[var(--color-border)] bg-[var(--color-bg)] shrink-0">
      <h1 className="text-lg font-semibold text-[var(--color-text)]">
        {title}
      </h1>
      {children && <div className="flex items-center gap-2">{children}</div>}
    </header>
  );
}

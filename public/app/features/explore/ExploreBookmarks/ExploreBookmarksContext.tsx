import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react';

interface ExploreBookmarksContextValue {
  drawerOpen: boolean;
  targetExploreId: string | null;
  openDrawer: (exploreId: string) => void;
  closeDrawer: () => void;
}

const ExploreBookmarksContext = createContext<ExploreBookmarksContextValue | undefined>(undefined);

export function ExploreBookmarksContextProvider({ children }: { children: ReactNode }) {
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [targetExploreId, setTargetExploreId] = useState<string | null>(null);

  const openDrawer = useCallback((exploreId: string) => {
    setTargetExploreId(exploreId);
    setDrawerOpen(true);
  }, []);

  const closeDrawer = useCallback(() => {
    setDrawerOpen(false);
    setTargetExploreId(null);
  }, []);

  const value = useMemo(
    () => ({
      drawerOpen,
      targetExploreId,
      openDrawer,
      closeDrawer,
    }),
    [closeDrawer, drawerOpen, openDrawer, targetExploreId]
  );

  return <ExploreBookmarksContext.Provider value={value}>{children}</ExploreBookmarksContext.Provider>;
}

export function useExploreBookmarksContext() {
  const context = useContext(ExploreBookmarksContext);
  if (!context) {
    throw new Error('useExploreBookmarksContext must be used within ExploreBookmarksContextProvider');
  }
  return context;
}

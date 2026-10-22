import React, { useContext } from "react";
import { createContext } from "react";

const ModalContext = createContext<{
  setModel: (model: React.ReactNode | null) => void;
}>({
  setModel: () => {
    throw new Error("add a ModelContextProvider to your hierarchy");
  },
});

export const ModalContextProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const [modal, setModals] = React.useState<React.ReactNode | null>([]);

  // Only the (state-stable) setter lives in the context value, and it is
  // memoized once, so opening/closing a modal re-renders the provider (which
  // renders {modal}) but NOT every useShowModal() consumer. A fresh value
  // object per render previously re-rendered the whole app on every
  // modal open/close.
  const value = React.useMemo(() => ({ setModel: setModals }), [setModals]);

  return (
    <ModalContext.Provider value={value}>
      {modal}
      {children}
    </ModalContext.Provider>
  );
};

export const useShowModal = () => {
  const context = useContext(ModalContext);
  return context.setModel;
};

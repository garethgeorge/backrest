import { Spin } from "antd";
import React, { useContext } from "react";
import { createContext } from "react";

const ModalContext = createContext<{
  model: React.ReactNode | null;
  setModel: (model: React.ReactNode | null) => void;
}>({
  model: null,
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

  return (
    <ModalContext.Provider
      value={{
        model: modal,
        setModel: setModals,
      }}
    >
      {modal}
      {children}
    </ModalContext.Provider>
  );
};

export const useShowModal = () => {
  const context = useContext(ModalContext);
  return context.setModel;
};

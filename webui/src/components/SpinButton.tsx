import React from "react";
import { Button, ButtonProps } from "antd";
import { useState } from "react";

export const SpinButton: React.FC<ButtonProps & {
  onClickAsync: () => Promise<void>;
}> = ({ onClickAsync, ...props }) => {
  const [loading, setLoading] = useState(false);

  const onClick = async () => {
    if (loading) {
      return;
    }
    try {
      setLoading(true);
      await onClickAsync();
    } finally {
      setLoading(false);
    }
  };

  return (
    <Button
      {...props}
      loading={loading}
      onClick={onClick}
    />
  );
}

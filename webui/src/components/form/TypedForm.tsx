import React, { useContext, useEffect, useMemo } from "react";
import { Form } from "antd";
import {
  clone,
  create,
  DescMessage,
  Message,
  MessageShape,
} from "@bufbuild/protobuf";
import {
  Plan,
  PlanSchema,
  Repo,
  RepoSchema,
} from "../../../gen/ts/v1/config_pb";

export type Paths<T> = T extends object
  ? {
      [K in Exclude<keyof T, symbol>]: `${K}${Paths<T[K]> extends never
        ? ""
        : `.${Paths<T[K]>}`}`;
    }[Exclude<keyof T, symbol>]
  : never;

export type DeepIndex<T, K extends string> = T extends object
  ? K extends `${infer F}.${infer R}`
    ? DeepIndex<Idx<T, F>, R>
    : Idx<T, K>
  : never;

type Idx<T, K extends string> = K extends keyof T ? T[K] : never;

const getValue = <T extends {}, K extends string>(
  obj: T,
  path: K
): DeepIndex<T, K> => {
  let curr: any = obj;
  const parts = path.split(".");
  for (let i = 0; i < parts.length; i++) {
    curr = curr[parts[i]];
  }
  return curr as DeepIndex<T, K>;
};

const setValue = <T extends {}, K extends Paths<T>>(
  obj: T,
  path: K,
  value: DeepIndex<T, K>
): void => {
  let curr: any = obj;
  const parts = path.split(".");
  for (let i = 0; i < parts.length - 1; i++) {
    curr = curr[parts[i]];
  }
  curr[parts[parts.length - 1]] = value;
};

interface typedFormCtx {
  formData: any;
  setFormData: (value: any) => any;
}

const TypedFormCtx = React.createContext<typedFormCtx>({
  formData: {},
  setFormData: () => {},
});

export const TypedForm = <Desc extends DescMessage>({
  schema,
  initialValue,
  onChange,
  children,
  ...props
}: {
  schema: Desc;
  initialValue: MessageShape<Desc>;
  children?: React.ReactNode;
  onChange?: (value: MessageShape<Desc>) => void;
}) => {
  const [form] = Form.useForm();
  const [formData, setFormData] =
    React.useState<MessageShape<Desc>>(initialValue);

  return (
    <TypedFormCtx.Provider value={{ formData, setFormData }}>
      <Form
        autoComplete="off"
        form={form}
        onFieldsChange={(changedFields, allFields) => {
          setFormData((prev) => {
            const next = clone(schema, prev);
            changedFields.forEach((field) => {
              setValue(next, field.name, field.value);
            });
            onChange?.(next);
            return next;
          });
        }}
        {...props}
      >
        {children}
      </Form>
    </TypedFormCtx.Provider>
  );
};

export const TypedFormItem = <T extends {}>({
  field,
  children,
  ...props
}: {
  field: Paths<T>;
  children?: React.ReactNode;
  props?: any;
}): React.ReactElement => {
  const { formData } = useContext(TypedFormCtx);

  return (
    <Form.Item
      name={field as string}
      initialValue={getValue(formData, field)}
      {...props}
    >
      {children}
    </Form.Item>
  );
};

export const TypedFormOneof = <T extends {}, K extends Paths<T>>({
  field,
  items,
}: {
  field: K;
  items: { [K2 in DeepIndex<T, K>]: React.ReactNode };
}): React.ReactNode => {
  const { formData } = useContext(TypedFormCtx);
  const c = getValue(formData as T, field);
  const val = items[c];
  if (!val) {
    return <></>;
  }
  return val;
};

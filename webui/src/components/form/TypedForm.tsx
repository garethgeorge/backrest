import React, { useContext, useEffect, useMemo } from "react";
import { Form } from "antd";
import {
  clone,
  create,
  DescMessage,
  Message,
  MessageShape,
} from "@bufbuild/protobuf";
import { useWatch } from "antd/es/form/Form";

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
  formData: Message;
  schema: DescMessage;
<<<<<<< Updated upstream
  setFormData: (value: any) => any;
=======
  setFormData: (fn: (prev: any) => any) => void;
>>>>>>> Stashed changes
}

const TypedFormCtx = React.createContext<typedFormCtx | null>(null);

<<<<<<< Updated upstream
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
} & {
  [key: string]: any;
}) => {
  const [form] = Form.useForm();
  const [formData, setFormData] =
    React.useState<MessageShape<Desc>>(initialValue);

  return (
    <TypedFormCtx.Provider value={{ formData, setFormData, schema }}>
      <Form
        autoComplete="off"
        form={form}
        {...props}
      >
=======
export const TypedForm = <T extends Message>({
  schema,
  formData,
  setFormData,
  children,
  ...props
}: {
  schema: DescMessage;
  formData: T;
  setFormData: (fn: (prev: T) => T) => void;
  children?: React.ReactNode;
} & {
  [key: string]: any;
}) => {
  const [form] = Form.useForm<T>();

  return (
    <TypedFormCtx.Provider value={{ formData, setFormData, schema }}>
      <Form autoComplete="off" form={form} {...props}>
>>>>>>> Stashed changes
        {children}
      </Form>
    </TypedFormCtx.Provider>
  );
};

export const TypedFormItem = <T extends Message>({
  field,
  children,
  ...props
}: {
  field: Paths<T>;
  children?: React.ReactNode;
} & {
  [key: string]: any;
}): React.ReactElement => {
  const { formData, setFormData, schema } = useContext(TypedFormCtx)!;
<<<<<<< Updated upstream
  const value = useWatch({ name: field as string });

  useEffect(() => {
=======
  const value = useWatch(field as string);

  useEffect(() => {
    console.log("set", field, value);
>>>>>>> Stashed changes
    setFormData((prev: any) => {
      const next = clone(schema, prev) as any as T;
      setValue(next, field, value);
      return next;
    });
  }, [value]);

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

interface oneofCase<T extends {}, F extends Paths<T>> {
  case: DeepIndex<T, `${F}.case`>;
  create: () => DeepIndex<T, `${F}.value`>;
  view: React.ReactNode;
}

<<<<<<< Updated upstream
export const TypedFormOneof = <T extends {}, F extends Paths<T>>({
=======
export const TypedFormOneof = <T extends Message, F extends Paths<T>>({
>>>>>>> Stashed changes
  field,
  items,
}: {
  field: F;
  items: oneofCase<T, F>[];
}): React.ReactNode => {
  const { formData, setFormData, schema } = useContext(TypedFormCtx)!;
<<<<<<< Updated upstream
  const v = getValue(formData as T, `${field}.value`);
  const c = getValue(formData as T, `${field}.case`);
  useEffect(() => {
    for (const item of items) {
      if (item.case === c) {
        setFormData((prev) => {
          const next = clone(schema, prev) as any as T;
          setValue(next, `${field}.value` as Paths<T>, v);
          return next;
        }

        const nv = item.create();
        setValue(formData, field + ".value", nv);
      }
    }
=======
  const c = useWatch(`${field}.case`);
  useEffect(() => {
    for (const item of items) {
      if (item.case === c) {
        setFormData((prev: any) => {
          const next = clone(schema, prev as T) as T;
          setValue(next, field, {
            case: c,
            value: item.create(),
          } as any);
          return next;
        });
      }
    }
    throw new Error("case " + c + " not found");
>>>>>>> Stashed changes
  }, [c]);

  for (const item of items) {
    if (item.case === c) {
      return item.view;
    }
  }
<<<<<<< Updated upstream
=======

>>>>>>> Stashed changes
  return null;
};

import { FormInstance } from "antd";
import type { ValidateErrorEntity } from "rc-field-form/lib/interface";

export const validateForm = async <T>(form: FormInstance<T>) => {
  try {
    return await form.validateFields();
  } catch (e: any) {
    if (e.errorFields) {
      const firstError = (e as ValidateErrorEntity).errorFields?.[0]
        ?.errors?.[0];
      throw new Error(firstError);
    }
    throw e;
  }
};

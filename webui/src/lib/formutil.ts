import { FormInstance } from "antd";

export const validateForm = async <T>(form: FormInstance<T>) => {
  try {
    await form.validateFields();
    return sanitizeUndefinedFields(form.getFieldsValue());
  } catch (e: any) {
    if (e.errorFields) {
      const firstError = (e as any).errorFields?.[0]
        ?.errors?.[0];
      throw new Error(firstError);
    }
    throw e;
  }
};

const sanitizeUndefinedFields = <T>(obj: T): T => {
  if (typeof obj !== "object" || obj === null) {
    return obj;
  } else if (Array.isArray(obj)) {
    return obj.map(sanitizeUndefinedFields) as any;
  }

  const newObj: any = {};
  for (const key in obj) {
    if (obj[key] !== undefined) {
      newObj[key] = sanitizeUndefinedFields(obj[key]);
    }
  }
  return newObj;
}

// regex allows alphanumeric, underscore, dash, and dot
// this should be kept in sync with values permitted by SanitizeID on the backend
export const namePattern = /^[a-zA-Z0-9_\-\.]+$/;

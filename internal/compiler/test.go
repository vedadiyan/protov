
type User struct {
    Id *int64 `protobuf:"varint,1,opt,name=id,proto3,oneof" json:"id"`
    FirstName string `protobuf:"bytes,2,opt,name=first_name,proto3" json:"first_name"`
    LastName string `protobuf:"bytes,3,opt,name=last_name,proto3" json:"last_name"`
    Email string `protobuf:"bytes,4,opt,name=email,proto3" json:"email"`
    Phone string `protobuf:"bytes,5,opt,name=phone,proto3" json:"phone"`
    EmergencyContact *string `protobuf:"bytes,6,opt,name=emergency_contact,proto3,oneof" json:"emergency_contact"`
    DateOfBirth string `protobuf:"bytes,7,opt,name=date_of_birth,proto3" json:"date_of_birth"`
    ProfilePicture *string `protobuf:"bytes,8,opt,name=profile_picture,proto3,oneof" json:"profile_picture"`
    Password string `protobuf:"bytes,9,opt,name=password,proto3" json:"password"` 
}

func (x *User) New() protolizer.Reflected {
    return new(User)
}

func (x *User) Type() protolizer.Type {
    return *protolizer.CaptureTypeByName("dashboard.users.User")
}

func (x *User) Encode(field *protolizer.Field, buffer *bytes.Buffer) error {
    switch field.Tags.Protobuf.FieldNum {
            case 1: {
                    if x.Id == nil {
                        return nil
                    } 
                    data, err := protolizer.SignedNumberEncoder(int64(*x.Id), field)
                    defer protolizer.Dealloc(data)
                    if err != nil {
                        return err
                    }
                    data.WriteTo(buffer)
                    return nil

            }
            case 2: {
                    data, err := protolizer.StringEncoder(x.FirstName, field)
                    defer protolizer.Dealloc(data)
                    if err != nil {
                        return err
                    }
                    data.WriteTo(buffer)
                    return nil

            }
            case 3: {
                    data, err := protolizer.StringEncoder(x.LastName, field)
                    defer protolizer.Dealloc(data)
                    if err != nil {
                        return err
                    }
                    data.WriteTo(buffer)
                    return nil

            }
            case 4: {
                    data, err := protolizer.StringEncoder(x.Email, field)
                    defer protolizer.Dealloc(data)
                    if err != nil {
                        return err
                    }
                    data.WriteTo(buffer)
                    return nil

            }
            case 5: {
                    data, err := protolizer.StringEncoder(x.Phone, field)
                    defer protolizer.Dealloc(data)
                    if err != nil {
                        return err
                    }
                    data.WriteTo(buffer)
                    return nil

            }
            case 6: {
                    if x.EmergencyContact == nil {
                        return nil
                    }
                    data, err := protolizer.StringEncoder(*x.EmergencyContact, field)
                    defer protolizer.Dealloc(data)
                    if err != nil {
                        return err
                    }
                    data.WriteTo(buffer)
                    return nil

            }
            case 7: {
                    data, err := protolizer.StringEncoder(x.DateOfBirth, field)
                    defer protolizer.Dealloc(data)
                    if err != nil {
                        return err
                    }
                    data.WriteTo(buffer)
                    return nil

            }
            case 8: {
                    if x.ProfilePicture == nil {
                        return nil
                    }
                    data, err := protolizer.StringEncoder(*x.ProfilePicture, field)
                    defer protolizer.Dealloc(data)
                    if err != nil {
                        return err
                    }
                    data.WriteTo(buffer)
                    return nil

            }
            case 9: {
                    data, err := protolizer.StringEncoder(x.Password, field)
                    defer protolizer.Dealloc(data)
                    if err != nil {
                        return err
                    }
                    data.WriteTo(buffer)
                    return nil

            }        
        default: {
            return fmt.Errorf("invalid field")
        }
    }    
} 

func (x *User) Decode(field *protolizer.Field, buffer *bytes.Buffer) error {
    switch field.Tags.Protobuf.FieldNum {
            case 1: { 
                    value, err := protolizer.SignedNumberDecoder(field, buffer)
                     if err != nil {
                        return err
                    } 
                    x.Id =&value
                     return nil
            }
            case 2: {
                    value, err := protolizer.StringDecoder(field, buffer)
                     if err != nil {
                        return err
                    } 
                    x.FirstName =value
                    return nil
            }
            case 3: {
                    value, err := protolizer.StringDecoder(field, buffer)
                     if err != nil {
                        return err
                    } 
                    x.LastName =value
                    return nil
            }
            case 4: {
                    value, err := protolizer.StringDecoder(field, buffer)
                     if err != nil {
                        return err
                    } 
                    x.Email =value
                    return nil
            }
            case 5: {
                    value, err := protolizer.StringDecoder(field, buffer)
                     if err != nil {
                        return err
                    } 
                    x.Phone =value
                    return nil
            }
            case 6: {
                    value, err := protolizer.StringDecoder(field, buffer)
                     if err != nil {
                        return err
                    } 
                    x.EmergencyContact =&value
                    return nil
            }
            case 7: {
                    value, err := protolizer.StringDecoder(field, buffer)
                     if err != nil {
                        return err
                    } 
                    x.DateOfBirth =value
                    return nil
            }
            case 8: {
                    value, err := protolizer.StringDecoder(field, buffer)
                     if err != nil {
                        return err
                    } 
                    x.ProfilePicture =&value
                    return nil
            }
            case 9: {
                    value, err := protolizer.StringDecoder(field, buffer)
                     if err != nil {
                        return err
                    } 
                    x.Password =value
                    return nil
            }   
        default: {
                return fmt.Errorf("invalid field")
        }     
    }    
} 

func (x *User) IsZero(field *protolizer.Field) bool {
    switch field.Tags.Protobuf.FieldNum {
            case 1: { 
                    return x.Id == nil 
            }
            case 2: {
                    return  len(x.FirstName) == 0
            }
            case 3: {
                    return  len(x.LastName) == 0
            }
            case 4: {
                    return  len(x.Email) == 0
            }
            case 5: {
                    return  len(x.Phone) == 0
            }
            case 6: {
                    return x.EmergencyContact == nil 
            }
            case 7: {
                    return  len(x.DateOfBirth) == 0
            }
            case 8: {
                    return x.ProfilePicture == nil 
            }
            case 9: {
                    return  len(x.Password) == 0
            }   
        default: {
           return true
        }     
    }    
}
type PostReq struct {
    User *User `protobuf:"bytes,1,opt,name=user,proto3" json:"user"` 
}

func (x *PostReq) New() protolizer.Reflected {
    return new(PostReq)
}

func (x *PostReq) Type() protolizer.Type {
    return *protolizer.CaptureTypeByName("dashboard.users.PostReq")
}

func (x *PostReq) Encode(field *protolizer.Field, buffer *bytes.Buffer) error {
    switch field.Tags.Protobuf.FieldNum {
            case 1: {
                    data, err := protolizer.FastMarshal(x.User)
                    if err != nil {
                        return err
                    }  
                    buffer.Write(data)

            }        
        default: {
            return fmt.Errorf("invalid field")
        }
    }    
} 

func (x *PostReq) Decode(field *protolizer.Field, buffer *bytes.Buffer) error {
    switch field.Tags.Protobuf.FieldNum {
            case 1: {
                    var value *User
                    bytes, err := protolizer.DecodeBytes(field, buffer)
                    if err != nil {
                        return err
                    }
                    err = protolizer.FastUnmarshal(value, bytes)
                     if err != nil {
                        return err
                    }     
                    x.User =value    
                    return nil
            }   
        default: {
                return fmt.Errorf("invalid field")
        }     
    }    
} 

func (x *PostReq) IsZero(field *protolizer.Field) bool {
    switch field.Tags.Protobuf.FieldNum {
            case 1: {
                    return false 
            }   
        default: {
           return true
        }     
    }    
}
type PostRes struct {
    Id int64 `protobuf:"varint,1,opt,name=id,proto3" json:"id"`
    FirstName string `protobuf:"bytes,2,opt,name=first_name,proto3" json:"first_name"`
    LastName string `protobuf:"bytes,3,opt,name=last_name,proto3" json:"last_name"` 
}

func (x *PostRes) New() protolizer.Reflected {
    return new(PostRes)
}

func (x *PostRes) Type() protolizer.Type {
    return *protolizer.CaptureTypeByName("dashboard.users.PostRes")
}

func (x *PostRes) Encode(field *protolizer.Field, buffer *bytes.Buffer) error {
    switch field.Tags.Protobuf.FieldNum {
            case 1: { 
                    data, err := protolizer.SignedNumberEncoder(int64(x.Id), field)
                    defer protolizer.Dealloc(data)
                    if err != nil {
                        return err
                    }
                    data.WriteTo(buffer)
                    return nil

            }
            case 2: {
                    data, err := protolizer.StringEncoder(x.FirstName, field)
                    defer protolizer.Dealloc(data)
                    if err != nil {
                        return err
                    }
                    data.WriteTo(buffer)
                    return nil

            }
            case 3: {
                    data, err := protolizer.StringEncoder(x.LastName, field)
                    defer protolizer.Dealloc(data)
                    if err != nil {
                        return err
                    }
                    data.WriteTo(buffer)
                    return nil

            }        
        default: {
            return fmt.Errorf("invalid field")
        }
    }    
} 

func (x *PostRes) Decode(field *protolizer.Field, buffer *bytes.Buffer) error {
    switch field.Tags.Protobuf.FieldNum {
            case 1: { 
                    value, err := protolizer.SignedNumberDecoder(field, buffer)
                     if err != nil {
                        return err
                    } 
                    x.Id =value
                     return nil
            }
            case 2: {
                    value, err := protolizer.StringDecoder(field, buffer)
                     if err != nil {
                        return err
                    } 
                    x.FirstName =value
                    return nil
            }
            case 3: {
                    value, err := protolizer.StringDecoder(field, buffer)
                     if err != nil {
                        return err
                    } 
                    x.LastName =value
                    return nil
            }   
        default: {
                return fmt.Errorf("invalid field")
        }     
    }    
} 

func (x *PostRes) IsZero(field *protolizer.Field) bool {
    switch field.Tags.Protobuf.FieldNum {
            case 1: { 
                    return  x.Id == 0
            }
            case 2: {
                    return  len(x.FirstName) == 0
            }
            case 3: {
                    return  len(x.LastName) == 0
            }   
        default: {
           return true
        }     
    }    
}
type GetReq struct {
    Id int64 `protobuf:"varint,1,opt,name=id,proto3" json:"id"` 
}

func (x *GetReq) New() protolizer.Reflected {
    return new(GetReq)
}

func (x *GetReq) Type() protolizer.Type {
    return *protolizer.CaptureTypeByName("dashboard.users.GetReq")
}

func (x *GetReq) Encode(field *protolizer.Field, buffer *bytes.Buffer) error {
    switch field.Tags.Protobuf.FieldNum {
            case 1: { 
                    data, err := protolizer.SignedNumberEncoder(int64(x.Id), field)
                    defer protolizer.Dealloc(data)
                    if err != nil {
                        return err
                    }
                    data.WriteTo(buffer)
                    return nil

            }        
        default: {
            return fmt.Errorf("invalid field")
        }
    }    
} 

func (x *GetReq) Decode(field *protolizer.Field, buffer *bytes.Buffer) error {
    switch field.Tags.Protobuf.FieldNum {
            case 1: { 
                    value, err := protolizer.SignedNumberDecoder(field, buffer)
                     if err != nil {
                        return err
                    } 
                    x.Id =value
                     return nil
            }   
        default: {
                return fmt.Errorf("invalid field")
        }     
    }    
} 

func (x *GetReq) IsZero(field *protolizer.Field) bool {
    switch field.Tags.Protobuf.FieldNum {
            case 1: { 
                    return  x.Id == 0
            }   
        default: {
           return true
        }     
    }    
}
type GetRes struct {
    User *User `protobuf:"bytes,1,opt,name=user,proto3" json:"user"` 
}

func (x *GetRes) New() protolizer.Reflected {
    return new(GetRes)
}

func (x *GetRes) Type() protolizer.Type {
    return *protolizer.CaptureTypeByName("dashboard.users.GetRes")
}

func (x *GetRes) Encode(field *protolizer.Field, buffer *bytes.Buffer) error {
    switch field.Tags.Protobuf.FieldNum {
            case 1: {
                    data, err := protolizer.FastMarshal(x.User)
                    if err != nil {
                        return err
                    }  
                    buffer.Write(data)

            }        
        default: {
            return fmt.Errorf("invalid field")
        }
    }    
} 

func (x *GetRes) Decode(field *protolizer.Field, buffer *bytes.Buffer) error {
    switch field.Tags.Protobuf.FieldNum {
            case 1: {
                    var value *User
                    bytes, err := protolizer.DecodeBytes(field, buffer)
                    if err != nil {
                        return err
                    }
                    err = protolizer.FastUnmarshal(value, bytes)
                     if err != nil {
                        return err
                    }     
                    x.User =value    
                    return nil
            }   
        default: {
                return fmt.Errorf("invalid field")
        }     
    }    
} 

func (x *GetRes) IsZero(field *protolizer.Field) bool {
    switch field.Tags.Protobuf.FieldNum {
            case 1: {
                    return false 
            }   
        default: {
           return true
        }     
    }    
}
type GetByEmailReq struct {
    Email string `protobuf:"bytes,1,opt,name=email,proto3" json:"email"` 
}

func (x *GetByEmailReq) New() protolizer.Reflected {
    return new(GetByEmailReq)
}

func (x *GetByEmailReq) Type() protolizer.Type {
    return *protolizer.CaptureTypeByName("dashboard.users.GetByEmailReq")
}

func (x *GetByEmailReq) Encode(field *protolizer.Field, buffer *bytes.Buffer) error {
    switch field.Tags.Protobuf.FieldNum {
            case 1: {
                    data, err := protolizer.StringEncoder(x.Email, field)
                    defer protolizer.Dealloc(data)
                    if err != nil {
                        return err
                    }
                    data.WriteTo(buffer)
                    return nil

            }        
        default: {
            return fmt.Errorf("invalid field")
        }
    }    
} 

func (x *GetByEmailReq) Decode(field *protolizer.Field, buffer *bytes.Buffer) error {
    switch field.Tags.Protobuf.FieldNum {
            case 1: {
                    value, err := protolizer.StringDecoder(field, buffer)
                     if err != nil {
                        return err
                    } 
                    x.Email =value
                    return nil
            }   
        default: {
                return fmt.Errorf("invalid field")
        }     
    }    
} 

func (x *GetByEmailReq) IsZero(field *protolizer.Field) bool {
    switch field.Tags.Protobuf.FieldNum {
            case 1: {
                    return  len(x.Email) == 0
            }   
        default: {
           return true
        }     
    }    
}
type GetByEmailRes struct {
    User *User `protobuf:"bytes,1,opt,name=user,proto3" json:"user"` 
}

func (x *GetByEmailRes) New() protolizer.Reflected {
    return new(GetByEmailRes)
}

func (x *GetByEmailRes) Type() protolizer.Type {
    return *protolizer.CaptureTypeByName("dashboard.users.GetByEmailRes")
}

func (x *GetByEmailRes) Encode(field *protolizer.Field, buffer *bytes.Buffer) error {
    switch field.Tags.Protobuf.FieldNum {
            case 1: {
                    data, err := protolizer.FastMarshal(x.User)
                    if err != nil {
                        return err
                    }  
                    buffer.Write(data)

            }        
        default: {
            return fmt.Errorf("invalid field")
        }
    }    
} 

func (x *GetByEmailRes) Decode(field *protolizer.Field, buffer *bytes.Buffer) error {
    switch field.Tags.Protobuf.FieldNum {
            case 1: {
                    var value *User
                    bytes, err := protolizer.DecodeBytes(field, buffer)
                    if err != nil {
                        return err
                    }
                    err = protolizer.FastUnmarshal(value, bytes)
                     if err != nil {
                        return err
                    }     
                    x.User =value    
                    return nil
            }   
        default: {
                return fmt.Errorf("invalid field")
        }     
    }    
} 

func (x *GetByEmailRes) IsZero(field *protolizer.Field) bool {
    switch field.Tags.Protobuf.FieldNum {
            case 1: {
                    return false 
            }   
        default: {
           return true
        }     
    }    
}